package com.dtrules.trace;

import com.dtrules.entity.IREntity;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;

public class Change {
    IREntity  e;
    RName     attribute;
    TraceNode execute_table;
    /**
     * @param changed
     * @param e
     * @param attribute
     */
    Change(IREntity e, RName attribute, TraceNode execute_table){
        this.e              = e;
        this.attribute      = attribute;
        this.execute_table  = execute_table;
    }
    
    @Override
    public int hashCode() {
        return attribute.hashCode();
    }
    
    @Override
    public boolean equals(Object arg0) {
        if(arg0 == null)                            return false;
        if(e.getID() != ((Change)arg0).e.getID())   return false;
        if(!attribute.equals((IRObject) attribute)) return false;
        return true;
    }
}
