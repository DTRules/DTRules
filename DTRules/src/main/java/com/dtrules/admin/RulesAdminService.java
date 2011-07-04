/** 
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
 **/
package com.dtrules.admin;

import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.OutputStream;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;

import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.IRSession;
import com.dtrules.session.RSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;

/**
 * @author Prasath Ramachandran
 * Feb 1, 2007
 * 
 * The RulesAdminService provides all the methods for adding to and maintaining a 
 * Rules in DTRules.  This is the interface to be used by editors and other tools
 * to add/remove decision tables, modify the Entity Description Dictionary, and 
 * create and modify Rule Sets.
 */
@SuppressWarnings("unchecked")
public class RulesAdminService implements IRulesAdminService{
	final private IRSession         session;
          private RulesDirectory   rd;          // The Rules Directory provides a place where Rule
                                                // sets are defined.
        
	/**
     * The RulesAdminService needs a session for specifying the output of debugging
     * and trace information.  It needs a Rules Directory to administer.  
	 */
    public RulesAdminService(final RSession session, final RulesDirectory rd) {
        super();
        this.session = session;
        this.rd      = rd;
    }

    /**
     * The RulesAdminService needs a session for specifying the output of debugging
     * and trace information.  It needs a Rules Directory to administer.  
     */
    public RulesAdminService(final IRSession session, final RulesDirectory rd) {
        super();
        this.session = session;
        this.rd      = rd;
    }
    
    
    
    /**
     * Create a new Attribute;  The Entity modified is specified by the attribute.
     * @param rulesetname The name of the RuleSet where the attribute should be created.
     * @param attribute   The REntityEntry defines the Entity to be modified.  It is
     *                    assumed that this Entity already exists.
     * @return true if the attribute was successfully added. 
	 */
	public boolean createAttribute(final String rulesetname, final REntityEntry attribute) {
        try {
            RuleSet  rs     = getRuleset(rulesetname);
            IREntity entity = rs.getEntityFactory(session).findRefEntity(attribute.attribute);
            entity.addAttribute(attribute);
            return true;
        } catch (RulesException e) {
            return false;
        }
		
	}

    /**
     * Given a Rule Set and an Entity, returns the list of EntityEntry objects
     * that define the meta information about each attribute of the entity. If an
     * error occurs while building the list, an empty list is returned.
     * 
     * @param rulesetname   The Rule Set in which the entity is defined
     * @param entityname    The entity name of intrest.
     * @return              The list of Entity Entries that define the attributes of the Entity
     */
    public List<REntityEntry> getEntityEntries(String rulesetname, String entityname){
        ArrayList<REntityEntry> entries = new ArrayList<REntityEntry>();
        RuleSet                 rs      = getRuleset(rulesetname);
        IREntity entity;
        try {
            entity = rs.getEntityFactory(session).findRefEntity(RName.getRName(entityname));
            Iterator<RName>         attribs = entity.getAttributeIterator();
            
            while(attribs.hasNext()){
                entries.add(entity.getEntry(attribs.next()));
            }
        } catch (RulesException e) { }
        
        return entries;
    }

    /**
     * Create a new Decision Table
     */
	public RDecisionTable createDecisionTable(String rulesetname, String decisiontablename) 
            throws RulesException{
		RuleSet rs     = getRuleset(rulesetname);
        return rs.getEntityFactory(session).newDecisionTable(decisiontablename,session);
	}

	/**
     * Creates a new Entity of the given name within the RuleSet.  If
     * the Entity already exists or the name is invalid, or this operation
     * fails in any other way, a null is returned.
	 * @see com.dtrules.admin.IRulesAdminService#createEntity(java.lang.String, java.lang.String)
     * @param rulesetname The name of the RuleSet in which to create this entity
     * @param EntityName  The name of the new Entity
	 */
	public IREntity createEntity(String rulesetname, boolean readonly, String entityName) {
        RName    name = RName.getRName(entityName);
		IREntity e    = session.getEntityFactory().findRefEntity(name);
		if(e!=null)return null;
        try {
            return session.getEntityFactory().findcreateRefEntity(readonly, name);
        } catch (RulesException e1) {
            return null;
        }
	}

	/**
     * Create a new Rule Set
	 * @see com.dtrules.admin.IRulesAdminService#createRuleset(com.dtrules.session.RuleSet)
	 */
	public void createRuleset(RuleSet ruleset) throws RulesException{
		rd.addRuleSet(ruleset);
	}

	/* (non-Javadoc)
	 * @see com.dtrules.admin.IRulesAdminService#getAttributes(java.lang.String, java.lang.String)
	 */
	@SuppressWarnings("unchecked")
	public List getAttributes(String rulesetname, String entityName) {
        if(rd==null)return null;
        try {
            RuleSet rs = rd.getRuleSet(RName.getRName(rulesetname));
            Iterator it = rs.getEntityFactory(session)
                          .findRefEntity(RName.getRName(entityName)).getAttributeIterator();
            return getList(it);
        } catch (RulesException e) {
            return null;
        }
	}

	/* (non-Javadoc)
	 * @see com.dtrules.admin.IRulesAdminService#getDecisionTable(java.lang.String, java.lang.String)
	 */
	public RDecisionTable getDecisionTable(String rulesetname, String DecisionTableName) {
        try {
            RuleSet rs = rd.getRuleSet(RName.getRName(rulesetname));
            return rs.getEntityFactory(session)
                   .findTable(RName.getRName(DecisionTableName));
        } catch (RulesException e) {
            return null;
        } 
	}

	/* (non-Javadoc)
	 * @see com.dtrules.admin.IRulesAdminService#getDecisionTables(java.lang.String)
	 */
	public List getDecisionTables(String rulesetname) {
		if(rd==null)return null;
        try {
            RuleSet rs = rd.getRuleSet(RName.getRName(rulesetname));
            Iterator it = rs.getEntityFactory(session).getDecisionTableRNameIterator();
            return getList(it);
        } catch (RulesException e) {
            return null;
        }
	}
	
    @SuppressWarnings("unchecked")
    private List getList(Iterator iterator){
        List list = new ArrayList();
        while(iterator.hasNext()){
            list.add(((IRObject)iterator.next()).stringValue());
        }
        return list;
        
    }
    
	/**
     * Return the list of Entities known to the given ruleset
     * @return List of Entity RName objects
	 * @see com.dtrules.admin.IRulesAdminService#getEntities(java.lang.String)
	 */
	public List getEntities(String rulesetname) {
        if(rd==null)return null;
        try {
            Iterator e = rd.getRuleSet(RName.getRName(rulesetname)).getEntityFactory(session)
                            .getEntityRNameIterator(); 
            return getList(e);
        } catch (RulesException e) {}
		return null;
	}

	/**
     * Return the RuleSet associated with the given RuleSet Name
     * @return RuleSet A set of Decision Tables and Entities which define a RuleSet
	 * @see com.dtrules.admin.IRulesAdminService#getRuleset(java.lang.String)
	 */
	public RuleSet getRuleset(String RulesetName) {
		if(rd==null)return null;
        return rd.getRuleSet(RName.getRName(RulesetName));
	}

	/**
     * Returns the list of RuleSets known to this Rules Directory
     * @return A List of RName objects giving the list Rule Sets
	 * @see com.dtrules.admin.IRulesAdminService#getRulesets()
	 */
	public List<String> getRulesets() {
		List<String> rulesets = new ArrayList<String>();
		for(Iterator it = rd.getRulesets().keySet().iterator(); it.hasNext();){
			RName name = (RName)it.next();
			rulesets.add(name.stringValue());
		}
		return rulesets;
	}

	/**
     * Loads the XML defining the RulesDirectory.  The given path must be a
     * path and filename which is valid in the file system, or a valid Java 
     * resource, or a valid URL.
     * @param pathtoDT 
	 * @see com.dtrules.admin.IRulesAdminService#initialize(java.lang.String)
	 */
	public void initialize(String pathtoDTrules) {
		String path = "C:\\eclipse\\workspace\\EB_POC\\CA HCO Plan\\xml\\";
		String file = path + "DTRules.xml";
		if(pathtoDTrules != null){
			file = pathtoDTrules;
		}
		
		try {
			rd = new RulesDirectory("",file);
			
		} catch (Exception e) {
			e.printStackTrace();
		}		
		
	}

	/**
     * This method changes (potentially) the attributes of an attribute of an
     * Entity.  The REntityEntry attribute must be populated correctly.
     * 
     * @param rulesetname    The Ruleset which holds the Entity for which the 
     *                       attribute applies.
     * @param attribute      Contains the Entity Name and all the other meta data                      
     *                       for the attribute to be changed.
     *                       
	 * @see com.dtrules.admin.IRulesAdminService#updateAttribute(java.lang.String, com.dtrules.entity.REntityEntry)
	 */
	public void updateAttribute(REntity entity, String rulesetname, REntityEntry attribute) throws RulesException {
        entity.addAttribute(attribute);
        
	}
    /**
     * Updates the Decision Table XML specified in the RuleSet with the Decision Tables as 
     * currently held in memory.  
     */
    public void saveDecisionTables(RSession session, String rulesetname) throws RulesException {
        RuleSet       rs       = rd.getRuleSet(RName.getRName(rulesetname));
        String        filepath = rs.getFilepath();
        String        filename = rs.getDT_XMLName();
        
        OutputStream out;
        try {
            out = new FileOutputStream(filepath+filename);
        } catch (FileNotFoundException e) {
            throw new RulesException(
                    "OutputFileCreationError",
                    "saveDecisionTables",
                    "An error occured openning the file: "+filepath+filename+"\n"+e
                   );
        }
        saveDecisionTables(out,session,rulesetname);
    }
    
    /**
     * Writes out the Decision Tables to the given output stream as the currently exist in 
     * Memory.
     * 
     * @param out outputstream to write the decision tables to.
     * @param session The session which gives us the outputstreams for debug statements and trace statements
     * @param rulesetname The Rule Set to be written out to the output stream
     * @throws RulesException
     */
	public void saveDecisionTables(OutputStream out, RSession session, String rulesetname) throws RulesException {
	    RuleSet       rs       = rd.getRuleSet(RName.getRName(rulesetname));
        EntityFactory ef       = rs.getEntityFactory(session);
        
        PrintStream  ps  = new PrintStream(out);
        Iterator<RName> tables = ef.getDecisionTableRNameIterator();
        ps.println("<decision_tables>");
        while(tables.hasNext()){
            RName dtname = tables.next();
            ef.getDecisionTable(dtname).writeXML(ps);
        }
        ps.println("\n</decision_tables>");
        
	}
    
    /**
     * Test function.
     * @param args ignored.
     */
    public static void main(String args[]){
        String path = "C:\\eclipse\\workspace\\EB_POC\\new_york_EB\\xml\\";
        String file = path + "DTRules.xml";    
    
        RulesDirectory    rd      = new RulesDirectory("",file);
        RuleSet           rs      = rd.getRuleSet(RName.getRName("ebdemo"));
        RSession          session;
        try {
            OutputStream  out     = new FileOutputStream("c:\\temp\\dt.xml");
                          session = new RSession(rs);
            RulesAdminService admin   = new RulesAdminService(session, rd);
                          
            admin.saveDecisionTables(out, session, "ebdemo");
        } catch (Exception e1) {
            System.out.println(e1.toString());
        }
    }
	/* (non-Javadoc)
	 * @see com.dtrules.admin.IRulesAdminService#updateEntity(java.lang.String, com.dtrules.entity.REntity)
	 */
	public void saveEDD(RSession session, String rulesetname) throws RulesException {
        RuleSet       rs       = rd.getRuleSet(RName.getRName(rulesetname));
        EntityFactory ef       = rs.getEntityFactory(session);
        String        filepath = rs.getFilepath();
        String        filename = rs.getEDD_XMLName();
        
        OutputStream out;
        try {
            out = new FileOutputStream(filepath+filename);
        } catch (FileNotFoundException e) {
            throw new RulesException(
                    "OutputFileCreationError",
                    "saveDecisionTables",
                    "An error occured openning the file: "+filepath+filename+"\n"+e
                   );
        }
        PrintStream  ps  = new PrintStream(out);
        Iterator<RName> entities = ef.getEntityRNameIterator();
        ps.println("<entity_data_dictionary>");
        while(entities.hasNext()){
            RName ename = entities.next();
            ef.findRefEntity(ename).writeXML(ps);
        }
        ps.println("\n</entity_data_dictionary>");
	}

	/* (non-Javadoc)
	 * @see com.dtrules.admin.IRulesAdminService#updateRuleset(com.dtrules.session.RuleSet)
	 */
	public void updateRuleset(RuleSet ruleset) {
		
	}
	
    
     /**
     * Given an incomplete table, this method returns a balanced table.
     * Entries in the table provided can be 'y', 'n', '-', or ' '.  If
     * blank, then it is assumed that the caller assumes this should be
     * flushed out. If the result will be larger than 16 columns, an 
     * null is returned. (You can't have a table larger than 16 columns).
     * @param table  -- A partial truth table.
     * @return
     */
    public String[][] balanceTable(String [] [] table){
        int rows = table.length;
        if(rows>4||rows<=0)return null;
        int cols = 1<<(rows);
        
        String newTable[][] = new String [rows][cols];
        
        for(int i=0; i< cols; i++){
            for(int j=0;j<rows; j++){
                newTable[j][i]= ((i>>(rows-1-j))&1)==1?"n":"y"; 
            }
        }
        
        return newTable;
    }

    public final RulesDirectory getRd() {
        return rd;
    }

    public final IRSession getSession() {
        return session;
    }
    
}
