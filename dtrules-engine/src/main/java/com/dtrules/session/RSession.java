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

package com.dtrules.session;


import java.io.PrintStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Set;

import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RTable;
import com.dtrules.interpreter.RType;
import com.dtrules.interpreter.operators.ROperator;
import com.dtrules.xmlparser.IXMLPrinter;
import com.dtrules.mapping.DataMap;
import com.dtrules.mapping.Mapping;

public class RSession implements IRSession {

    final RuleSet               rs;
	final DTState               dtstate;
    EntityFactory               ef;
    private int                 uniqueID;
    HashMap<Object,IREntity>    entityInstances = new HashMap<Object,IREntity>();
    IComputeDefaultValue        ComputeDefault = new ComputeDefaultValue();
    ArrayList<DataMap>          registeredMaps = new ArrayList<DataMap>();  
    IDateParser					dateParser = new DateParser();
    
    boolean                     printIds = true;
    
    public IDateParser getDateParser() {
		return dateParser;
	}

	public void setDateParser(IDateParser dateParser) {
		this.dateParser = dateParser;
	}

	/**
     * Get the list of Data Maps Registered to this session.
     * @return
     */
    public ArrayList<DataMap> getRegisteredMaps(){
        return registeredMaps;
    }

    private void registerMap(DataMap datamap){
        if(!registeredMaps.contains(datamap)){
            registeredMaps.add(datamap);
        }
    }
    
    /**
     * Allocate a registered data map.  If you want to map Data Objects
     * into the Rules Engine, you need to use this call, providing the
     * mapping which provides information about these Data Objects.
     * @return
     */
    @SuppressWarnings("deprecation")
    public DataMap getDataMap(Mapping map, String tag){
        DataMap datamap = new DataMap(this, map,tag,null);
        registerMap(datamap);
        return datamap;
    }
    
    
    /**
     * Get the default mapping
     * @return
     */
    public Mapping getMapping () {
        return rs.getMapping(this);
    }
    
    /**
     * Get a named mapping file
     * @param filename
     * @return
     */
    public Mapping getMapping (String filename){
        return rs.getMapping(filename, this);
    }
    
    /**
     * Gets the Rules Directory used to create this session
     * @return The Rules Directory used to create this session
     */
    public RulesDirectory getRulesDirectory() {
        return rs.getRulesDirectory();
    }
    
    /**
     * Allocate a new compiler from the Rule Set and return it.
     * Returns null if no compiler is available, or any error is 
     * thrown in the process.
     */
    public ICompiler getCompiler() {
        try {
        	if(rs.getDefaultCompiler()!=null){
        		ICompiler c = rs.getDefaultCompiler().newInstance();
        		try {
					c.setSession(this);
	        		return c;
				} catch (Exception e) { }
        	}
		} catch (IllegalAccessException e) {
		} catch (InstantiationException e) {
		} catch (RulesException e) {
		}
		return null;
    }

    /**
     * Each RSession is associated with a particular RuleSet
     * @param _ef
     * @throws RulesException
     */
    public RSession(RuleSet _rs) throws RulesException {
        rs      = _rs;
        dtstate  = new DTState(this);
        ef       = _rs.getEntityFactory(this);
        uniqueID = ef.getUniqueID()+10000;      // Put some distance between the reference entity ids 
        										// and the instances we create.
        /**
         * Add all the reference entities to the session list 
         * of entities.
         */
        Iterator<RName> ie = ef.referenceEntities.keySet().iterator();
        while(ie.hasNext()){
            IREntity e  = ef.findRefEntity(ie.next());
            String   id = e.getID()+"";
            if(entityInstances.containsKey(id)){
                throw new RulesException("duplicate","new RSession()","Duplicate id "+id+" found between:\n"
                        +e.getName()+" and "+entityInstances.get(id).getName());
            }
            entityInstances.put(id,e);
        }
        try { 
            dtstate.entitypush(ROperator.getPrimitives()); 
            dtstate.entitypush(ef.decisiontables);
        } catch (RulesException e) {
            throw new RulesException("Initialization Error", 
                    "RSession", 
                    "Failed to initialize dtstate in init(): "+e.toString());
        }
    }
    
    /**
     * Does a recursive dump of the given entity to standard out.
     * This is a debugging thing.  If DEBUG isn't set in the State of the Session,
     * calling this routine does nothing at all.
     * @param e
     */
    public void dump(REntity e) throws RulesException {

        if(!dtstate.testState(DTState.DEBUG))return;    // Leave if nothing to do.
        dtstate.traceTagBegin("entity", "name",e.getName().stringValue(),"id=",e.getID()+"");
        dump(e,1); 
        dtstate.traceTagEnd();
    }
    
    private HashMap<IREntity,ArrayList<IREntity>> boundries = new HashMap<IREntity,ArrayList<IREntity>>(); // Track printing Entity from Entity boundries to stop recursive printing.
    
    private String getType(IREntity e, RName n) throws RulesException{
        RType type = e.getEntry(n).type;
        return type.toString();
    }
        
    /**
     * Dumps the Entity and its attributes to the debug output source.  However,
     * if debug isn't enabled, the routine does absolutely nothing.
     * @param e The Entity to be dumped.
     * @param depth Dumping is a private, recursive thing.  depth helps us track its recursive nature.
     */
    private void dump(IREntity e,int depth){
        Iterator<RName> anames = e.getAttributeIterator();
        while(anames.hasNext()){
            try {
                RName        aname = anames.next();
                IRObject     value = e.get(aname);
                
                dtstate.traceTagBegin("attribute", "name",aname.stringValue(),"type",getType(e,aname));
                int type = e.getEntry(aname).type.getId();
                
               if(type == IRObject.iEntity) {
                      if(value.type().getId() == IRObject.iNull){
                          dtstate.traceInfo("value","type","null","value","null",null);
                      }else{
	                      dtstate.traceTagBegin("entity", 
	                              "name",   ((REntity)value).getName().stringValue(),
	                              "id",     printIds ? ((REntity)value).getID()+"" : "");
	                      
	                      if(!(boundries.get(e)!= null && boundries.get(e).contains(value))){
	                          dtstate.debug(" recurse\n");
	                      }else{
	                          if(boundries.get(e)==null)boundries.put(e, new ArrayList<IREntity>());
	                          boundries.get(e).add(value.rEntityValue());
	                          dump((IREntity)value, depth+1);
	                      }
	                      dtstate.traceTagEnd();
                      }
               }else if (type == IRObject.iArray) {
                      List<IRObject> values = value.arrayValue();
                      for(IRObject v : values){
                          if(v.type().getId() ==IRObject.iEntity){
                              dump((REntity)v,depth+2);
                          }else{
                              dtstate.traceInfo("value","v",v.stringValue(),null);
                          }
                      }
               } else { 
                       dtstate.traceInfo("value","v",value.stringValue(),null);
               }
                
                dtstate.traceTagEnd();
            } catch (RulesException e1) {
                dtstate.debug("Rules Engine Exception\n");
                e1.printStackTrace(dtstate.getErrorOut());
            }
        }
    }
    
    
    /**
     * Return an ID that is unique for this session.  The ID is generated using a simple
     * counter.  This ID will not be unique across all sessions, nor is it very likely to
     * be unique when compared to IDs generated with by methods.
     * <br><br>
     * Unique IDs are used to distinguish various instances of entities apart during
     * exectuion and/or during the generation or loading of a Trace file. They can be 
     * used for other purposes as well, under the assumption that the number of unique IDs
     * are less than or equal to 0x7FFFFFFF hex or 2147483647 decimal.
     * 
     * @return a Unique ID generated by a simple counter. 
     */
    public int getUniqueID(){
        return uniqueID++;
    }	
	

	/**
     * Compiles the given string into an executable array, per the Rules Engine's interpreter.
     * Assuming this compile completes, the given array is executed.  A RulesException can be
     * thrown either by the compile of the string, or the execution of the resulting array.
     * @param s String to be compiled.
     * @exception RulesException thrown if any problem occurs compiling or executing the string.
     * @see com.dtrules.session.IRSession#execute(java.lang.String)
     */
	public void execute(String s) throws RulesException {
		try {
            RString.newRString(s,true).execute(dtstate);
        } catch (RulesException e) {
            if(getState().testState(DTState.TRACE)){
                getState().traceInfo("Error", e.toString());
                getState().traceEnd();
            }
            throw e;
        }
		return;
	}

	public void initialize(String entrypoint) throws RulesException {
        // First make sure our context is up to snuff (by checking that all of our
        // required entities are in the current context.
        List<String> entities = rs.contexts.get(entrypoint);
        if (entities == null){
            throw new RulesException("undefined","executeAt","The entry point '"
                    +entrypoint+"' is undefined");
        }
        for(String entity : entities){
            if(!dtstate.inContext(entity)){
                dtstate.entitypush(createEntity(null, entity));
            }
        }	    
	}
	
	@Override
    public void executeAt(String entrypoint) throws RulesException {
	    initialize(entrypoint);                            // Set up our context
	    String dtname = rs.entrypoints.get(entrypoint);    // Now get our entry point, and execute!
	    ef.findDecisionTable(RName.getRName(dtname)).execute(dtstate);
	}
    
	/**
     * Returns the state object for this Session.  The state is used by the Rules Engine in
     * the interpretation of the decision tables.
     * @return DTState The object holds the Data Stack, Entity Stack, and other state of the Rules Engine. 
	 */
    public DTState getState(){return dtstate; }

	/**
	 * @return the ef
	 */
	public EntityFactory getEntityFactory() {
		return ef;
	}
	/**
	 * 
	 */
    public void setEntityFactory(EntityFactory ef){
    	this.ef = ef;
    }

	/**
     * Create an Instance of an Entity of the given name.
     * @param name The name of the Entity to create
     * @return     The entity created.
     * @throws RulesException Thrown if any problem occurs (such as an undefined Entity name)
	 */
	public IREntity createEntity(Object id, String name) throws RulesException{
        RName entity = RName.getRName(name);
		return createEntity(id, entity);
	}

    /**
     * Create an Instance of an Entity of the given name.
     * @param name The name of the Entity to create
     * @return     The entity created.
     * @throws RulesException Thrown if any problem occurs (such as an undefined Entity name)
     */
	public IREntity createEntity(Object id, RName name) throws RulesException{
	    if(id==null){
	        id = getUniqueID()+"";
	    }
		IREntity ref = ef.findRefEntity(name);
        if(ref==null){
            throw new RulesException("undefined","session.createEntity","An attempt ws made to create the entity "+name.stringValue()+"\n" +
                    "This entity isn't defined in the EDD");
        }
   
        if(ref.isReadOnly()) return ref;
        
        if(getState().testState(DTState.TRACE)){
        	IREntity e = entityInstances.get(id);
            if(e==null){
            	
            	try { // Try and make the uniqueId match.  But ignore errors.
					int eid = Integer.parseInt(id.toString());
					if(uniqueID!=eid) uniqueID = eid;
				} catch (NumberFormatException e1) {
					e1.printStackTrace();
				}

            	e = (REntity) ref.clone(this);
            	
            	dtstate.traceInfo(
            			"createentity",
            			"name", e.getName().stringValue(),
            			"id",   ""+e.getID(),
            			null);

            	entityInstances.put(id,e);
            }
            return e;
    	}else{
    		REntity e = (REntity) ref.clone(this);
    		try {
                e.setId(Integer.parseInt(id.toString()));
            } catch (NumberFormatException e1) { }
    		return e;
    	}
   
	}
	
	
	/**
	 * @return the rs
	 */
	public RuleSet getRuleSet() {
		return rs;
	}
	
	/**
	 * Entity Printing Routines
	 */
	
	
	 public void printEntity(IXMLPrinter rpt, String tag, IREntity e) throws Exception {
	     if(tag==null)tag = e.getName().stringValue();
	     IRObject id = e.get(RName.getRName("mapping*key"));
	     String   idString = id!=null?id.stringValue():"--none--";
         rpt.opentag(tag,"DTRulesId",e.getID()+"","id",printIds ? idString : "");
         Set<RName> names = e.getAttributeSet();
         RName keys[] = sort(names);
         for(RName name : keys){
             IRObject v    = e.get(name);
             if(v.type().getId()==IRObject.iArray && v.rArrayValue().size()==0) continue;
             String   vstr = v==null?"":v.stringValue();
             rpt.printdata("attribute","name",name.stringValue(), vstr);
         }
	 }

 
	 public void printArray(IXMLPrinter rpt, ArrayList<IRObject> entitypath, ArrayList<IRObject> printed, DTState state, String name, RArray rarray)throws RulesException{
         if(name!=null && name.length()>0){
             rpt.opentag(name, "id", printIds ? rarray.getID():"");
         }else{
             rpt.opentag("array", "id",printIds ? rarray.getID():"");
         }
         for(IRObject element : rarray){
             printIRObject(rpt, entitypath, printed, state,"",element);
         }
         rpt.closetag();
     }

	   public void printTable(
	           IXMLPrinter rpt, 
	           ArrayList<IRObject> entitypath, 
	           ArrayList<IRObject> printed, 
	           DTState state, 
	           String name, 
	           RTable table)throws RulesException{
	       
	       Set<IRObject> keys = table.getTable().keySet();
	         rpt.opentag("map", "id", printIds ? table.getId(): "");
    	         for(IRObject key : keys){
    	            IRObject value = table.getValue(key);
    	            rpt.opentag("pair");
    	                rpt.opentag("key");
    	                    printIRObject(rpt,entitypath,printed,state,"",key);
    	                rpt.closetag();
    	                rpt.opentag("value");
    	                    printIRObject(rpt, entitypath, printed, state, "", value);
    	                rpt.closetag();
    	            rpt.closetag();
    	         }
           rpt.closetag();
	 
	   }

	 
     public void printEntityReport(IXMLPrinter rpt, DTState state, String objname ) {
         printEntityReport(rpt,false, false,state,objname);
     }
 
     public void printEntityReport(IXMLPrinter rpt, boolean printIds, boolean verbose, DTState state, String name ) {
         this.printIds = printIds;
    	 IRObject obj = state.find(name);
         printEntityReport(rpt,printIds, verbose,state,name,obj);
     }
     
     public void printEntityReport(IXMLPrinter rpt, boolean printIds, boolean verbose, DTState state, String name, IRObject obj ) {
         this.printIds = printIds;
         ArrayList<IRObject> entitypath = new ArrayList<IRObject>();
         ArrayList<IRObject> printed = new ArrayList<IRObject>();
         try {
             if(obj==null){
                 rpt.printdata("unknown", "object", name,null);
             }else{
                 printIRObject(rpt,entitypath, printed,state,name,obj);
             }    
         } catch (RulesException e) {
             rpt.print_error(e.toString());
         }
     }
          
     public void printIRObject(IXMLPrinter rpt, ArrayList<IRObject> entitypath, ArrayList<IRObject> printed, DTState state, String name, IRObject v) throws RulesException {
    	 int otype = v.type().getId();
         if(otype == IRObject.iTable) {
                 if(name.length()!=0)rpt.opentag(name);
                 printTable(rpt,entitypath,printed,state,name,v.rTableValue());
                 if(name.length()!=0)rpt.closetag();
         } else if( otype == IRObject.iEntity){
                 if(name.length()!=0)rpt.opentag(name);
                 printAllEntities(rpt, entitypath, printed, state, v.rEntityValue());
                 if(name.length()!=0)rpt.closetag();
         } else if( otype == IRObject.iArray) {
                 if(name.length()!=0)rpt.opentag(name);
                 printArray(rpt, entitypath, printed, state, name, v.rArrayValue());
                 if(name.length()!=0)rpt.closetag();
         } else {
                 String   vstr = v==null?"":v.stringValue();
                 String tname = name.replaceAll("[*]", "_");
                 if(tname.length()==0)tname = "value";
                 rpt.printdata(tname, "name", name, vstr);
         }
     } 
     
    public void printAllEntities(IXMLPrinter rpt, ArrayList<IRObject> entitypath, ArrayList<IRObject> printed, DTState state, IREntity e) throws RulesException   {
         String entityName = e.getName().stringValue();
         if(entitypath.contains(e) && entitypath.get(entitypath.size()-1)==e){
                 rpt.printdata(entityName, "self reference");
         }else if (printed!= null && printed.contains(e)){
                 rpt.printdata(entityName,"DTRulesId",printIds ? e.getID():"","id",e.get("mapping*key").stringValue(),"multiple reference");  
         }else{
             entitypath.add(e);
             if(printed!=null) printed.add(e);
             IRObject id = e.get(RName.getRName("mapping*key"));
             String   idString = id!=null?id.stringValue():"--none--";
             rpt.opentag(entityName,"DTRulesId",printIds ? e.getID():"","id",idString);
             Set<RName> keys = e.getAttributeSet();
             RName akeys [] = sort(keys);
             for(RName name : akeys){
                 IRObject v    = e.get(name);
                 printIRObject(rpt, entitypath, printed, state, name.stringValue(), v);
             }
             rpt.closetag();
             entitypath.remove(entitypath.size()-1);
         }
     }
    
     public RName [] sort(Set<RName> keys) {
    	 RName  ret[] = keys.toArray(new RName[0]);
    	 int cnt = ret.length;
    	 for(int i = 0; i< cnt-1; i++){
    		 for(int j = 0; j < cnt-1-i; j++){
    			 RName one = ret[j];
    			 RName two = ret[j+1];
    			 if(one.stringValue().compareTo(two.stringValue())>0){
    				 RName hld = ret[j];
    				 ret[j]    = ret[j+1];
    				 ret[j+1]  = hld;
    			 }
    		 }
    	 }
    	 return ret;
     }
    
	/**
	 * Prints all the balanced form of all the decision tables in this session to
	 * the given output stream 
	 * @throws RulesException
	 */
    public void printBalancedTables(PrintStream out)throws RulesException {
        Iterator<RName> dts = this.getEntityFactory().getDecisionTableRNameIterator();
        while(dts.hasNext()){
            RName dtname = dts.next();
            RDecisionTable dt = this.getEntityFactory().findDecisionTable(dtname);
            String t;
            try{
            	t = dt.getBalancedTable().getPrintableTable();
            }catch(RulesException e){
            	System.out.flush();
            	System.err.println("The Decision Table '"+dt.getName().stringValue()
            			+"' is too complex, and must be split into two tables.");
            	t = "Table too Big to Print";
            	System.err.flush();
            }
            out.println();
            out.println(dtname.stringValue());
            out.println();
            out.println(t);
        }
    }
    /**
     * This is the method that defines how the default value in the EDD is converted into a Rules Engine object
     * @return
     */
    public IComputeDefaultValue getComputeDefault() {
        return ComputeDefault;
    }
    /**
     * This is the method that defines how the default value in the EDD is converted into a Rules Engine object
     * @return
     */
    public void setComputeDefault(IComputeDefaultValue computeDefault) {
        ComputeDefault = computeDefault;
    }
    
}
