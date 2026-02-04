/*
 * Copyright 2004-2009 DTRules.com, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.dtrules.compiler.el;

import java.io.PrintStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;

import org.antlr.v4.runtime.CharStreams;
import org.antlr.v4.runtime.CommonTokenStream;
import org.antlr.v4.runtime.BaseErrorListener;
import org.antlr.v4.runtime.RecognitionException;
import org.antlr.v4.runtime.Recognizer;

import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.ICompiler;
import com.dtrules.session.IRSession;
import com.dtrules.session.IRType;

/**
 * ANTLR 4 based EL Compiler implementation.
 * This replaces the JFlex/CUP based EL compiler with ANTLR 4.
 */
public class ELAntlr implements ICompiler {

    private HashMap<RName, IRType> types = null;
    private EntityFactory ef;
    private IRSession session;

    private int localcnt = 0;
    private HashMap<String, ELTypeResolver.RLocalType> localtypes = new HashMap<String, ELTypeResolver.RLocalType>();

    public void setTableName(String tablename) {
        // Not used in this implementation
    }

    /**
     * Add an identifier and type to the types HashMap.
     */
    private void addType(IREntity entity, RName name, int itype) throws Exception {
        ELType type = (ELType) types.get(name);
        if (type == null) {
            type = new ELType(name, itype, entity);
            types.put(name, type);
        } else {
            if (type.getType() != itype) {
                String entitylist = entity.getName().stringValue();
                for (int i = 0; i < type.getEntities().size(); i++) {
                    entitylist += " " + type.getEntities().get(i).getName().stringValue();
                }
                throw new Exception("Conflicting types for attribute " + name + " on " + entitylist);
            }
            type.addEntityAttribute(entity);
        }
    }

    /**
     * Get all the types out of the Entity Factory and make them
     * available to the compiler.
     */
    public HashMap<RName, IRType> getTypes(EntityFactory ef) throws Exception {
        if (types != null) return types;

        types = new HashMap<RName, IRType>();
        Iterator<RName> entities = ef.getEntityRNameIterator();
        while (entities.hasNext()) {
            RName name = entities.next();
            IREntity entity = ef.findRefEntity(name);
            Iterator<RName> attribs = entity.getAttributeIterator();
            addType(entity, entity.getName(), IRObject.iEntity);
            while (attribs.hasNext()) {
                RName attribname = attribs.next();
                REntityEntry entry = entity.getEntry(attribname);
                addType(entity, attribname, entry.type.getId());
            }
        }

        Iterator<RName> tables = ef.getDecisionTableRNameIterator();
        while (tables.hasNext()) {
            RName tablename = tables.next();
            ELType type = new ELType(tablename, IRObject.iDecisiontable, (REntity) ef.getDecisiontables());
            if (types.containsKey(tablename)) {
                System.out.println("Multiple Decision Tables found with the name '" + types.get(tablename) + "'");
            }
            types.put(tablename, type);
        }

        return types;
    }

    /**
     * Prints all the types known to the compiler
     */
    public void printTypes(PrintStream out) throws RulesException {
        Object typenames[] = types.keySet().toArray();
        for (int i = 0; i < typenames.length - 1; i++) {
            for (int j = 0; j < typenames.length - 1; j++) {
                RName one = (RName) typenames[j], two = (RName) typenames[j + 1];
                if (one.compare(two) > 0) {
                    Object hold = typenames[j];
                    typenames[j] = typenames[j + 1];
                    typenames[j + 1] = hold;
                }
            }
        }
        for (int i = 0; i < typenames.length - 1; i++) {
            for (int j = 0; j < typenames.length - 1; j++) {
                RName one = (RName) typenames[j], two = (RName) typenames[j + 1];
                if (((ELType) types.get(one)).getType() > ((ELType) types.get(two)).getType()) {
                    Object hold = typenames[j];
                    typenames[j] = typenames[j + 1];
                    typenames[j + 1] = hold;
                }
            }
        }
        for (int i = 0; i < typenames.length; i++) {
            out.println(types.get(typenames[i]));
        }
    }

    private ArrayList<String> parsedTokens = new ArrayList<String>();

    public ArrayList<String> getParsedTokens() {
        return parsedTokens;
    }

    public void newDecisionTable() {
        localcnt = 0;
        localtypes.clear();
    }

    /**
     * Custom error listener for ANTLR that throws RulesException
     */
    private static class ELErrorListener extends BaseErrorListener {
        private final String source;
        private RulesException error;

        public ELErrorListener(String source) {
            this.source = source;
        }

        @Override
        public void syntaxError(Recognizer<?, ?> recognizer, Object offendingSymbol,
                                int line, int charPositionInLine,
                                String msg, RecognitionException e) {
            String location = "Line " + line + ":" + charPositionInLine;
            String before = "";
            String after = "";

            String[] lines = source.split("\n");
            if (line > 0 && line <= lines.length) {
                String currentLine = lines[line - 1];
                if (charPositionInLine < currentLine.length()) {
                    before = currentLine.substring(0, charPositionInLine);
                    after = currentLine.substring(charPositionInLine);
                } else {
                    before = currentLine;
                }
            }

            error = new RulesException("compileError", location, before + " *ERROR*=> " + after + "\n" + msg);
        }

        public RulesException getError() {
            return error;
        }
    }

    /**
     * The actual routine to compile either an action or condition.
     */
    private String compile(String s) throws RulesException {
        parsedTokens.clear();

        // Add trailing semicolon to match TokenFilter behavior (it injects SEMI at EOF)
        if (!s.trim().endsWith(";")) {
            s = s + " ;";
        }

        // Create lexer and parser
        ELLexer lexer = new ELLexer(CharStreams.fromString(s));
        CommonTokenStream tokens = new CommonTokenStream(lexer);
        ELParser parser = new ELParser(tokens);

        // Set up error handling
        ELErrorListener errorListener = new ELErrorListener(s);
        lexer.removeErrorListeners();
        lexer.addErrorListener(errorListener);
        parser.removeErrorListeners();
        parser.addErrorListener(errorListener);

        // Parse
        ELParser.DoneContext tree = parser.done();

        // Check for errors
        if (errorListener.getError() != null) {
            throw errorListener.getError();
        }

        // Create type resolver and visitor
        ELTypeResolver typeResolver = new ELTypeResolver(session, types, localtypes);
        ELCompilerVisitor visitor = new ELCompilerVisitor(typeResolver);
        visitor.setLocalCnt(localcnt);

        // Visit the parse tree to generate postfix
        String result = visitor.visit(tree);

        // Update local variable count
        localcnt = visitor.getLocalCnt();

        return result;
    }

    public String getLastPreFixExp() {
        return "";
    }

    /**
     * Build a compiler instance.
     */
    public ELAntlr() {
    }

    /**
     * @param session Needed to generate the symbol table used by the compiler.
     */
    public void setSession(IRSession session) throws Exception {
        this.session = session;
        ef = session.getEntityFactory();
        getTypes(ef);
    }

    @Override
    public String compileContext(String context) throws RulesException {
        return compile("context " + context);
    }

    @Override
    public String compileAction(String action) throws RulesException {
        return compile("action " + action);
    }

    @Override
    public String compileInitialAction(String action) throws RulesException {
        return compile("action " + action);
    }

    @Override
    public String compileCondition(String condition) throws RulesException {
        return compile("condition " + condition);
    }

    /**
     * Returns the compiled version of the policy statement.
     */
    @Override
    public String compilePolicyStatement(String policyStatement) throws RulesException {
        if (policyStatement == null) return "";
        policyStatement = policyStatement.replaceAll("\"", "'");
        StringBuilder sbuff = new StringBuilder();
        int s = 0;
        int e = policyStatement.indexOf("{", s);
        boolean first = true;
        while (e > 0) {
            sbuff.append("\"");
            sbuff.append(policyStatement.substring(s, e));
            if (first) {
                first = false;
                sbuff.append("\" ");
            } else {
                sbuff.append("\" s+ ");
            }
            s = e;
            e = policyStatement.indexOf("}", s);
            if (e < 0) {
                throw new RuntimeException("Unbalanced braces: " + policyStatement);
            }

            String source = "policystatement " + policyStatement.substring(s + 1, e);

            try {
                String value = compile(source);
                sbuff.append(value);
                sbuff.append("cvs strconcat ");
            } catch (Exception ex) {
                throw new RulesException("ParseError", "PolicyStatements", ex.toString() + "\n Source: >>" + source + "<<");
            }

            s = e + 1;
            e = policyStatement.indexOf("{", s);
        }

        sbuff.append("\"");
        sbuff.append(policyStatement.substring(s));
        if (s == 0) {
            sbuff.append("\" ");
        } else {
            sbuff.append("\" strconcat");
        }

        return sbuff.toString();
    }

    public HashMap<RName, IRType> getTypes() {
        return types;
    }

    /**
     * Return the list of Possibly (but we can't tell for sure) referenced attributes
     * so far by this compiler.
     */
    public ArrayList<String> getPossibleReferenced() {
        ArrayList<String> v = new ArrayList<String>();
        for (IRType type : types.values()) {
            v.addAll(((ELType) type).getPossibleReferenced());
        }
        return v;
    }

    /**
     * Return the list of UnReferenced attributes so far by this compiler
     */
    public ArrayList<String> getUnReferenced() {
        ArrayList<String> v = new ArrayList<String>();
        for (IRType type : types.values()) {
            v.addAll(((ELType) type).getUnReferenced());
        }
        return v;
    }
}
