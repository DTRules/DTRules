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
package com.dtrules.compiler.ebl;

import java.util.HashMap;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.session.IRType;

/**
 * Resolves identifier types for the ANTLR 4 EL parser.
 * Replaces the functionality of TokenFilter from the JFlex/CUP implementation.
 */
public class EBLTypeResolver {

    private final HashMap<RName, IRType> types;
    private final IRSession session;
    private final DTState state;
    private final HashMap<String, RLocalType> localtypes;

    public EBLTypeResolver(IRSession session, HashMap<RName, IRType> types, HashMap<String, RLocalType> localtypes) {
        this.types = types;
        this.session = session;
        this.state = session.getState();
        this.localtypes = localtypes;
    }

    /**
     * Local variable type information
     */
    public static class RLocalType {
        public String name;
        public int type;
        public int index;

        public RLocalType(String name, int type, int index) {
            this.name = name;
            this.type = type;
            this.index = index;
        }
    }

    /**
     * Result of resolving an identifier
     */
    public static class ResolvedIdentifier {
        public final int type;
        public final String value;
        public final String leftValue;
        public final boolean isLocal;
        public final String entityPrefix;

        public ResolvedIdentifier(int type, String value, String leftValue, boolean isLocal, String entityPrefix) {
            this.type = type;
            this.value = value;
            this.leftValue = leftValue;
            this.isLocal = isLocal;
            this.entityPrefix = entityPrefix;
        }
    }

    /**
     * Resolve an identifier and return its type information.
     *
     * @param ident The identifier to resolve
     * @return ResolvedIdentifier containing type and value information
     * @throws RulesException if the identifier cannot be resolved
     */
    public ResolvedIdentifier resolve(String ident) throws RulesException {
        String entity = "";
        String identPart = ident;

        // Handle entity.attribute notation
        if (ident.indexOf('.') >= 0) {
            entity = ident.substring(0, ident.indexOf('.'));
            identPart = ident.substring(ident.indexOf('.') + 1);
        }

        int theType = identType(identPart, entity);

        // Check for local variables
        if (theType == -1) {
            String lowerIdent = ident.toLowerCase();
            if (localtypes.containsKey(lowerIdent)) {
                RLocalType t = localtypes.get(lowerIdent);
                String value = t.index + " local@ ";
                String leftValue = t.index + " local! ";
                return new ResolvedIdentifier(t.type, value, leftValue, true, "");
            }
        }

        // Not found
        if (theType == -1) {
            return new ResolvedIdentifier(-1, ident, "/" + ident + " xdef ", false, entity);
        }

        // Found - construct value strings
        String fullIdent = (entity.length() > 0 ? entity + "." : "") + identPart;
        String value = fullIdent + " ";
        String leftValue = "/" + fullIdent + " xdef ";

        return new ResolvedIdentifier(theType, value, leftValue, false, entity);
    }

    /**
     * Resolve a possessive (e.g., "client's")
     *
     * @param possessive The possessive string (with 's)
     * @return ResolvedIdentifier for the entity
     * @throws RulesException if not an entity type
     */
    public ResolvedIdentifier resolvePossessive(String possessive) throws RulesException {
        String ident = possessive.replace("'s", "");
        String entity = "";

        if (ident.indexOf('.') >= 0) {
            entity = ident.substring(0, ident.indexOf('.'));
            ident = ident.substring(ident.indexOf('.') + 1);
        }

        int theType = identType(ident, entity);

        // Check if it's a local entity
        if (theType == -1) {
            String lowerIdent = ident.toLowerCase();
            if (localtypes.containsKey(lowerIdent)) {
                RLocalType t = localtypes.get(lowerIdent);
                theType = t.type;
                if (theType != IRObject.iEntity) {
                    throw new RulesException("InvalidPossessive", "parser",
                            "The Local Variable " + ident + " is not an Entity");
                }
                String value = t.index + " local@ ";
                return new ResolvedIdentifier(theType, value, null, true, "");
            }
        }

        if (theType != IRObject.iEntity && theType != -1) {
            throw new RulesException("InvalidPossessive", "parser",
                    "Invalid Possessive. " + ident + " is not an Entity");
        }

        if (theType == -1) {
            throw new RulesException("UndefinedAttribute", "parser",
                    "Undefined attribute '" + possessive + "'");
        }

        String fullIdent = (entity.length() > 0 ? entity + "." : "") + ident;
        return new ResolvedIdentifier(theType, fullIdent + " ", null, false, entity);
    }

    /**
     * Look up an identifier in the symbol table and return its type.
     */
    private int identType(String ident, String entity) {
        ELType type = (ELType) types.get(RName.getRName(ident, true));
        boolean defined = true;

        if (entity != null && entity.length() != 0 && type != null) {
            defined = false;
            for (IREntity rEntity : type.getEntities()) {
                if (entity.equalsIgnoreCase(rEntity.getName().stringValue())) {
                    if (rEntity.containsAttribute(RName.getRName(ident))) {
                        defined = true;
                    }
                    break;
                }
            }
        }

        if (type == null || !defined) {
            RName rname = RName.getRName(
                    (entity == null || entity.length() == 0 ? "" : entity + ".") + ident);
            try {
                IRObject o = session.getState().find(rname);
                if (o == null) return -1;
                return o.type().getId();
            } catch (RulesException e) {
                return -1;
            }
        }

        type.addRef(entity);
        return type.getType();
    }

    /**
     * Add a new local variable
     *
     * @param name Variable name
     * @param type Variable type (from IRObject constants)
     * @param index Variable index
     * @return The initialization postfix code
     */
    public String addLocal(String name, int type, int index, String initialValue) {
        name = name.toLowerCase();
        RLocalType t = new RLocalType(name, type, index);
        localtypes.put(name, t);

        if (initialValue != null) {
            return initialValue + "allocate execute deallocate pop ";
        } else {
            return "null allocate execute deallocate pop ";
        }
    }

    /**
     * Check if identifier is already defined as local
     */
    public boolean isLocalDefined(String name) {
        return localtypes.containsKey(name.toLowerCase());
    }

    /**
     * Get local variable info
     */
    public RLocalType getLocal(String name) {
        return localtypes.get(name.toLowerCase());
    }

    /**
     * Get type constant name for debugging
     */
    public static String getTypeName(int type) {
        if (type == IRObject.iBoolean) return "BOOLEAN";
        if (type == IRObject.iString) return "STRING";
        if (type == IRObject.iInteger) return "INTEGER";
        if (type == IRObject.iDouble) return "DOUBLE";
        if (type == IRObject.iEntity) return "ENTITY";
        if (type == IRObject.iName) return "NAME";
        if (type == IRObject.iArray) return "ARRAY";
        if (type == IRObject.iDecisiontable) return "DECISIONTABLE";
        if (type == IRObject.iDate) return "DATE";
        if (type == IRObject.iTable) return "TABLE";
        if (type == IRObject.iOperator) return "OPERATOR";
        if (type == IRObject.iXmlValue) return "XMLVALUE";
        if (type == IRObject.iNull) return "NULL";
        return "UNKNOWN(" + type + ")";
    }
}
