package com.dtrules.samples.bookpreview.datamodel;

import com.dtrules.mapping.DataMap;
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
    public void write2DataMap(DataMap datamap) {
        datamap.opentag(this,"page");
        datamap.readDO(this,"page");
        datamap.closetag();
    }
    
}
