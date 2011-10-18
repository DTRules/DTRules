package com.dtrules.trace;

import com.dtrules.entity.IREntity;

public class Change {
    IREntity  e;
    String    attribute;
    boolean   changed;
    TraceNode whereChanged;
    /**
     * @param changed
     * @param e
     * @param attribute
     */
    Change(boolean changed, IREntity e, String attribute, TraceNode whereChanged){
        this.e              = e;
        this.attribute      = attribute;
        this.changed        = changed;
        this.whereChanged   = whereChanged;
    }
    
    @Override
    public boolean equals(Object arg0) {
        if(arg0 == null)                 return false;
        if(e != ((Change)arg0).e)        return false;
        if(!attribute.equals(attribute)) return false;
        return true;
    }
}
