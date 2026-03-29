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

package com.dtrules.automapping.access;

import java.lang.reflect.Method;
import java.lang.reflect.ParameterizedType;
import java.lang.reflect.Type;
import java.util.ArrayList;
import java.util.List;
import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.AutoDataMapDef;
import com.dtrules.automapping.Group;
import com.dtrules.automapping.Label;
import com.dtrules.automapping.MapType;
import com.dtrules.automapping.nodes.MapNodeAttribute;
import com.dtrules.automapping.nodes.MapNodeList;
import com.dtrules.automapping.nodes.MapNodeMap;
import com.dtrules.automapping.nodes.MapNodeObject;
import com.dtrules.automapping.nodes.MapNodeRef;

/**
 * @author Paul Snow
 *
 */
public class JavaSource implements IDataSource {
    final AutoDataMapDef  autoDataMapDef;
    final String          type = "java";
    
    public JavaSource(AutoDataMapDef autoDataMapDef){
        this.autoDataMapDef = autoDataMapDef;
    }

    @Override
    public Label createLabel(
            AutoDataMap autoDataMap, 
            Group  group,
            String labelName, 
            String key, 
            boolean singular, 
            Object object) {
        if (key == null || key.trim().length()==0) key = "";
        if (labelName == null || labelName.trim().length()== 0){
            throw new RuntimeException("A Label has to have a valid Label Name.");
        }
        Label labelObj = group.findLabel(labelName,"java",object.getClass().getName());
        if(labelObj == null ){
            labelObj = Label.newLabel(group,labelName,object.getClass().getName(),key,singular);
        }
        
        if(labelObj.isCached()== false){
            cacheGetters(group, labelObj,object);
            labelObj.setCached(true);
        }
        return labelObj;
    }

    @Override
    public String getSpec(Object obj) {
        return obj.getClass().getName();
    }
    
    @Override
    public String getName(Object obj) {
        String name = obj.getClass().getSimpleName();
        name = name.substring(0,1).toLowerCase() +
               (name.length()>1? name.substring(1): "");
        return name;
    }

    /**
     * Looks for an accessor that looks like getClassNameId()
     */
    @Override
    public String getKey(Object obj) {
        String name = obj.getClass().getSimpleName();
        Method[] methods = obj.getClass().getMethods();
        for(Method method : methods){
            String mName = method.getName();
            String propertyName = getterName(method);
            if(propertyName != null
               && mName.substring(3).startsWith(name) 
               && propertyName.length()-2 == name.length()
               && propertyName.endsWith("Id")){
                return propertyName;
            }
        }
        return "";
    }
    
    /**
     * Extracts the property name from a Getter Name.  Of course, this can't
     * be done if this method really isn't a getter.  If the method isn't a
     * getter, a null is returned.
     * @param method
     * @return
     */
    public static String getterName(Method method){
        boolean startsWithGet = method.getName().startsWith("get");
        boolean startsWithIs  = method.getName().startsWith("is");
        Class<?>v             = method.getReturnType();
        boolean v_void        = void.class.equals(v);
        boolean v_boolean     = boolean.class.equals(v) || Boolean.class.equals(v);
        int     paramCnt      = method.getParameterTypes().length;        
        // All getters must past the following tests!
        if(!startsWithGet && !startsWithIs)                 return null;
        if(startsWithGet && method.getName().length()==3)   return null;
        if(startsWithIs  && method.getName().length()==2)   return null;
        if(startsWithIs && !v_boolean)                      return null;
        if(v_void)                                          return null;
        if(paramCnt != 0)                                   return null;
        // Okay!  This is a Getter! Extract the name!
        String name = method.getName().substring(startsWithGet ? 3:2); // get and is are my two options
        name = name.substring(0,1).toLowerCase()+                      // lowercase first char 
               (name.length()>1 ? name.substring(1):"");               // If more char follow, add them
        return name;
    }
    
    /**
     * Extracts the property name from a Setter Name.  Of course, this can't
     * be done if this method really isn't a setter.  If the method isn't a
     * setter, a null is returned.
     * @param method
     * @return
     */
    public static String setterName(Method method){
        boolean startsWithSet = method.getName().startsWith("set");
        Class<?>v             = method.getReturnType();
        boolean v_void        = void.class.equals(v);
        int     paramCnt      = method.getParameterTypes().length;        
        // All setters must past the following tests!
        if(!startsWithSet)				                    return null;
        if(method.getName().length()==3)                    return null;
        if(!v_void)                                         return null;
        if(paramCnt != 1)                                   return null;
        // Okay!  This is a Setter! Extract the name!
        String name = method.getName().substring(3); 	
        name = name.substring(0,1).toLowerCase()+                      // lowercase first char 
               (name.length()>1 ? name.substring(1):"");               // If more char follow, add them
        return name;
    }
     
    public static class Accessor {
        Boolean   caseSensitive = true;    // name is case sensitive
        String    name;                    // Property name
        Method    getter;                  // Getter method
        Method    setter;                  // Setter method
        Class<?>  typeClass;               // The type of the property
        MapType   type;                    // Return/Parameter type
        String    typeText;                // Actual text of the type
        MapType   subType=MapType.NULL;    // With lists and references, you have a subtype
        String    subTypeText="";          // Actual text of the subtype
        
        @Override
        public String toString(){
            return name;
        }
        
        Accessor(String name, Class<?> type, Class<?> subtype){
            this.name = name;
            this.typeClass = type;
            this.typeText = type.getSimpleName();
            this.type = MapType.get(typeText);
            if(subtype!=null){
                subTypeText = subtype.getSimpleName();
                subType = MapType.get(subTypeText);
            }
        }
    }

    /**
     * Look at the Java Object for this Label, and cache all the getters.  For
     * every getter, we look for a setter and cache it too.  We should be able
     * to handle setters without getters, but for now that case is ignored.
     */
    public void cacheGetters(Group group, Label label, Object object){
        try{
            Class<?> obj = object.getClass();                   // Get this object class
            if(obj==null)return;                                // None found?  Shouldn't happen.
            // Map all the attributes
            ArrayList<Accessor> accessors = getAccessors(group, object);
            for(Accessor accessor : accessors){
                try {                                           // We will ignore getters we can't access
                    JavaAttribute attribute = JavaAttribute.newAttribute(
                            label, 
                            accessor.name, 
                            accessor.getter.getName(), 
                            accessor.setter == null ? null : accessor.setter.getName(),
                            accessor.typeClass,
                            accessor.type,
                            accessor.typeText,
                            accessor.subType,
                            accessor.subTypeText);
                    attribute.setGetMethod(accessor.getter);
                    attribute.setSetMethod(accessor.setter);
                } catch (Exception e) {}
            }
        } catch (Exception e) {
            e.printStackTrace();
            System.out.println(e.toString());
        }
    }

 
    /**
     * We look at the return type specified on the Method to determine the subType
     * of a List.  For anything else, we are going to return a null for the subType.
     * @param method 
     * @return subType of list
     */
    private Class<?> getSubType(Method method){
        Type     returnType  = method.getGenericReturnType();
        Class<?> returnClass = null;
        if(returnType instanceof ParameterizedType){
            ParameterizedType type = (ParameterizedType) returnType;
            Type[] typeArguments = type.getActualTypeArguments();
            for(Type typeArgument : typeArguments){
                try{      
                    returnClass = (Class<?>) typeArgument;
                }catch(Exception e){
                    returnClass = Class.class;
                }
            }
        }
        return returnClass;
    }
    
    /**
     * Looks up all the getters off the given object.
     * @param obj
     * @return
     */
    public ArrayList<Accessor> getAccessors(Group group, Object obj) {
        
        ArrayList<Accessor> gs = new ArrayList<Accessor>();
        Class<?> javaclass  = obj.getClass();

        if(javaclass != null){
            Method methods[] = javaclass.getMethods();
            for(Method method : methods){
                String name = getterName(method);
                if(name!= null){
                    Class<?> subType = getSubType(method);
                    if(subType!=null){
                        Label label;
                        
                        if(subType.isPrimitive()){
                            label = group.findLabelBySpec(method.getName());
                        }else{
                            label = group.findLabelBySpec(subType.getName());
                        }
                        
                        if(label!=null && group.isPruned(label.getSpec())){
                            continue;
                        }
                    }
                    try{
                    	Method setter     = null;
                    	String settername = "set"+name.substring(0,1).toUpperCase()+
                    			((name.length()>1)?name.substring(1):"");
                    	for(Method s : methods){
                    		if(s.getName().equals(settername)) {
                    			setter = s;
                    			break;
                    		}
                    	}
                    	
                        Accessor getter = new Accessor(
                                name,
                                method.getReturnType(),
                                getSubType(method));
                        getter.getter = method;
                        getter.setter = setter;
                        getter.caseSensitive = Boolean.TRUE;
                        gs.add(getter);
                    }catch(Exception e){}
                }
            }
        }
        return gs;
    }
  
    @Override
    public List<Object> getChildren(Object obj){
        return new ArrayList<Object>();    
    }
    /**
     * We don't need this mechanism for Java Objects as it is pretty easy for us
     * to just go grab the key.
     */
    public Object getKeyValue(MapNodeObject node, Object object){
        return node.getKey();
    }
   
    @Override
    public void update(AutoDataMap autoDataMap, MapNodeAttribute node) {
    	if(node.getParent() instanceof MapNodeObject){
        	Object obj = ((MapNodeObject)node.getParent()).getSource();
    		if(obj != null){
    			node.getAttribute().set(obj, node.getData());
    		}
    	}
        
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeList node) {
    	if(node.getParent() instanceof MapNodeObject){
        	Object obj = ((MapNodeObject)node.getParent()).getSource();
    		if(obj != null){
    			if(node.getList().size()==0){
            		node.getAttribute().set(obj,null);
            		return;
            	}	
    			node.getAttribute().set(obj, node.getList());
    		}
    	}
        
    }

	@Override
    public void update(AutoDataMap autoDataMap, MapNodeMap node) {
    	if(node.getParent() instanceof MapNodeObject){
        	Object obj = ((MapNodeObject)node.getParent()).getSource();
    		if(obj != null){
    			if(node.getMap().size()==0){
            		node.getAttribute().set(obj,null);
            		return;
            	}	
    			node.getAttribute().set(obj, node.getMap());
    		}
    	}
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeObject node) {
        
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeRef node) {
        
    }
    
    
}
