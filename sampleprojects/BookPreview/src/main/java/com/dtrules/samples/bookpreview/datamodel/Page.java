package com.dtrules.samples.bookpreview.datamodel;

import com.dtrules.xmlparser.XMLPrinter;

public class Page extends ABookObj {
    int     number;

    public int getNumber() {
        return number;
    }

    public void setNumber(int number) {
        this.number = number;
    }
    
    @Override
    void print(XMLPrinter xout) {
        
        xout.opentag("page","id",id);
        if(printed){
            xout.closetag();
            return;
        }
        xout.printdata("number",number);
        xout.closetag();
    }
    
}
