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

import com.dtrules.automapping.Label;
import com.dtrules.automapping.MapType;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * 
 * 
 * @author Paul Snow
 *
 */
public class JavaAttribute implements IAttribute {
    private String   name;                  // Property name
    private String   getter;                // Getter name
    private String   setter;                // Setter name
    private Method   getMethod=null;        // getMethod for accessing this attribute
    private Method   setMethod=null;        // getMethod for accessing this attribute
    private Class<?> typeClass;             // Class of the property
    private MapType  type;                  // This is the property type.  The return 
    private String   typeText;				//   type for a getter, or parameter type for a setter.
    private MapType  subType;               // If a List or direct reference to a class, we need the subtype.
    private String   subTypeText;
    private Label    label;                 // This is the label to which this attribute belongs.
    public String toString(){
        return name;
    }
    
    /* (non-Javadoc)
     * @see com.dtrules.automapping.access.IAttribute#getName()
     */
    public String getName() {
        return name;
    }
   
    @Override
    public boolean isKey() {
        return name.equals(label.getKey());
    }

    /**
     * @return the label
     */
    public Label getLabel() {
        return label;
    }

    /**
     * @param name the name to set
     */
    public void setName(String name) {
        this.name = name;
    }
    /**
     * @return the getter
     */
    public String getGetter() {
        return getter;
    }
    /**
     * @param getter the getter to set
     */
    public void setGetter(String getter) {
        this.getter = getter;
    }
    /**
     * @return the setter
     */
    public String getSetter() {
        return setter;
    }
    /**
     * @param setter the setter to set
     */
    public void setSetter(String setter) {
        this.setter = setter;
    }
    /**
     * @return the getMethod
     */
    public Method getGetMethod() {
        return getMethod;
    }
    /**
     * @param getMethod the getMethod to set
     */
    public void setGetMethod(Method getMethod) {
        this.getMethod = getMethod;
    }
    /**
     * @return the setMethod
     */
    public Method getSetMethod() {
        return setMethod;
    }
    /**
     * @param setMethod the setMethod to set
     */
    public void setSetMethod(Method setMethod) {
        this.setMethod = setMethod;
    }
    
   public Class<?> getTypeClass() {
		return typeClass;
	}

	public void setTypeClass(Class<?> typeClass) {
		this.typeClass = typeClass;
	}

	/* (non-Javadoc)
     * @see com.dtrules.automapping.access.IAttribute#getType()
     */
    public MapType getType() {
        return type;
    }
    /* (non-Javadoc)
     * @see com.dtrules.automapping.access.IAttribute#getTypeText()
     */
    public String getTypeText() {
        return typeText;
    }
    /**
     * @param typeText the typeText to set
     */
    public void setTypeText(String typeText) {
        this.typeText = typeText;
    }
    /* (non-Javadoc)
     * @see com.dtrules.automapping.access.IAttribute#getSubType()
     */
    public MapType getSubType() {
        return subType;
    }
    /**
     * @return the subType
     */
    public void setSubType(MapType subType) {
        this.subType = subType;
    }
    /* (non-Javadoc)
     * @see com.dtrules.automapping.access.IAttribute#getSubTypeText()
     */
    public String getSubTypeText() {
        return subTypeText;
    }
    /**
     * @param subTypeText the subTypeText to set
     */
    public void setSubTypeText(String subTypeText) {
        this.subTypeText = subTypeText;
    }
    
	public static JavaAttribute newAttribute(
            Label    label, 
            String   name, 
            String   getter, 
            String   setter,
            Class<?> typeClass,
            MapType  type){
        return newAttribute(label,name,getter,setter,typeClass,type,type.getName(),null,null);
    }
    /**
     * Either creates or updates an attribute.  So the "new" part of newAttribute
     * is a bit misleading.  You can update the getters or setters, but not the
     * type of an attribute.  This is because we do not have the assurance of 
     * symmetry between getters and setters (not every property has both), so they
     * are collected separately. 
     * @param label
     * @param name
     * @param getter
     * @param setter
     * @param type
     * @return
     */
	public static JavaAttribute newAttribute(
            Label    label, 
            String   name, 
            String   getter, 
            String   setter,
            Class<?> typeclass,
            MapType  type,
            String   typeText,
            MapType  subType,
            String   subTypeText){
    
        if(subTypeText == null) subTypeText = "";
        
        if(getter == null ){
            if(type == MapType.BOOLEAN){
                getter = "is"+name.substring(0,1).toUpperCase()+(name.length()>1?name.substring(1):"");
            }else{
                getter = "get"+name.substring(0,1).toUpperCase()+(name.length()>1?name.substring(1):"");
            }
        }
        if(setter == null ){
            setter = "set"+name.substring(0,1).toUpperCase()+(name.length()>1?name.substring(1):"");
            
        }
        JavaAttribute a = null;
        for(IAttribute a1 :  label.getAttributes()){
            JavaAttribute a2 = (JavaAttribute) a1;
            if (a2.name.equals(name)){
                a = a2;
                break;
            }
        }
        
        // We ignore nulls, but we don't allow any other sort of change to
        // the type of an attribute.
        if(a != null && type!=null && a.type != type){
            throw new RuntimeException("Cannot override attribute types");
        }
        
        if(a == null){                      // Every Attribute has to have a name
           a = new JavaAttribute();         //   and must be in a label's list.
           a.name      = name;
           a.label     = label; 
           label.getAttributes().add(a);
        }
        // We only change these attributes if the current one's are null.
        a.getter       = getter      == null ? a.getter      : getter;
        a.setter       = setter      == null ? a.setter      : setter;
        a.type         = type        == null ? a.type        : type;
        a.typeText     = typeText    == null ? a.typeText    : typeText;
        a.subType      = subType     == null ? a.subType     : subType;
        a.subTypeText  = subTypeText == null ? a.subTypeText : subTypeText;
        return a;
    }
    
    public Object get(Object obj){
        if(getMethod == null) return null;              // Should never happen. We cache elsewhere.
        try{
            return getMethod.invoke(obj,new Object[0]);
        }catch(Exception e){
            return null;
        }
    }

    /**
     * Set the given value on the given object.
     * @param obj
     * @param value
     */
    public void set(Object obj, Object value){
        if(setMethod == null )return;                   // Should never happen.  We cache elsewhere.
        Object arglist[] = new Object[1];
        arglist[0] = value;
        try {
			setMethod.invoke(obj, arglist);
		} catch (Exception e) {}
    }
    
    
    public void printXML(XMLPrinter xout){
        xout.printdata("attribute", 
                "name",   name,
                "getter", getter,
                "setter", setter,
                "type",   typeText,
                "subtype", subTypeText,
                null);
    }

}
