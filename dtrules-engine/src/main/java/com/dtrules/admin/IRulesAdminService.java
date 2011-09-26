/** 
 * Copyright 2004-2011 DTRules.com, Inc.
 * 
 * See http://DTRules.com for updates and documentation for the DTRules Rules Engine  
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

import java.util.List;

import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.RSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;

/**
 * @author Paul Snow
 * Feb 1, 2007
 *
 */
public interface IRulesAdminService {
    
	public void         initialize(String pathtoDTrules);
	public List<String> getRulesets(); // List of Ruleset Names
	public RuleSet      getRuleset(String RulesetName);
	public void         updateRuleset(RuleSet ruleset);
	
	public List<?>      getDecisionTables(String rulesetname);
	public RDecisionTable getDecisionTable(String rulesetname, String DecisionTableName);
	public void           saveDecisionTables(RSession session, String rulesetname) throws RulesException;
    public RDecisionTable createDecisionTable(String rulesetname, String decisiontablename) throws RulesException;	
    //public Node getDecisionTableTree(String rulesetname) --- not needed now

	public List     getEntities(String rulesetname);
	public List     getAttributes(String rulesetname, String entityName);
	public IREntity createEntity(String rulesetname, boolean readonly, String EntityName);
	public boolean  createAttribute(String rulesetname, REntityEntry attribute);
	public void     saveEDD(RSession session, String rulesetname)throws RulesException;
	public void     updateAttribute(REntity entity, String rulesetname,REntityEntry attribute)throws RulesException;
   
    /**
     * Given an incomplete table, this method returns a balanced table.
     * Entries in the table provided can be 'y', 'n', '-', or ' '.  If
     * blank, then it is assumed that the caller assumes this should be
     * flushed out. If the result will be larger than 16 columns, an 
     * null is returned. (You can't have a table larger than 16 columns).
     * @param table  -- A partial truth table.
     * @return
     */
    public String[][] balanceTable(String [] [] table);
    /**
     * Given a Rule Set and an Entity, returns the list of EntityEntry objects
     * that define the meta information about each attribute of the entity.
     * @param rulesetname   The Rule Set in which the entity is defined
     * @param entityname    The entity name of intrest.
     * @return              The list of Entity Entries that define the attributes of the Entity
     */
    public List<REntityEntry> getEntityEntries(String rulesetname, String entityname);

    public RulesDirectory getRulesDirectory();
}
