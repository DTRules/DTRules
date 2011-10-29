package com.dtrules.samples.bookpreview.datamodel;

import com.dtrules.mapping.DataMap;
import com.dtrules.xmlparser.XMLPrinter;

public class Publisher extends ABookObj {
    int     chapter_limit;
    int     page_limit;
    public int getChapter_limit() {
        return chapter_limit;
    }
    public void setChapter_limit(int chapter_limit) {
        this.chapter_limit = chapter_limit;
    }
    public int getPage_limit() {
        return page_limit;
    }
    public void setPage_limit(int page_limit) {
        this.page_limit = page_limit;
    }
    
    @Override
    public void write2DataMap(DataMap datamap) {
        datamap.opentag(this,"publisher");
        if(printed){
            datamap.closetag();
            return;
        }
        datamap.readDO(this,"publisher");
        datamap.closetag();
    }
    
}
