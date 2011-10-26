package com.dtrules.samples.bookpreview.datamodel;

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
    void print(XMLPrinter xout) {
        xout.opentag("publisher","id",id);
        if(printed){
            xout.closetag();
            return;
        }
        xout.printdata("chapter_limit",chapter_limit);
        xout.printdata("page_limit",page_limit);
        xout.closetag();
    }
    
}
