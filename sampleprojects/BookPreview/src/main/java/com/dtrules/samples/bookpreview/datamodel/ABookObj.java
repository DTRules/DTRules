package com.dtrules.samples.bookpreview.datamodel;

import com.dtrules.xmlparser.XMLPrinter;

public abstract class ABookObj {
    static  int     sid = 1;
    int             id  = sid++;
    boolean         printed = false;
    
    abstract void print(XMLPrinter xout);
    
    int getId() {return id;}
    
    public boolean getPrinted(){
        if(!printed) {
            printed = true;
            return false;        
        }
        return true;
    }
}
