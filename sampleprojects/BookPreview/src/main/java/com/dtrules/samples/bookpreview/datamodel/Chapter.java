package com.dtrules.samples.bookpreview.datamodel;

import com.dtrules.mapping.DataMap;
import com.dtrules.xmlparser.XMLPrinter;

public class Chapter extends ABookObj{
    int     begin_page;
    int     end_page;
    
    public int getBegin_page() {
        return begin_page;
    }
    public void setBegin_page(int begin_page) {
        this.begin_page = begin_page;
    }
    public int getEnd_page() {
        return end_page;
    }
    public void setEnd_page(int end_page) {
        this.end_page = end_page;
    }
    
    void print(DataMap datamap, String tag){
        datamap.opentag(this,tag);
        if(printed){
            datamap.closetag();
            return;
        }
        datamap.readDO(this, tag);
        datamap.closetag();
    }
    
    @Override
    public void print(DataMap datamap) { print(datamap, "chapter");}
    
    @Override
    public String toString() {
        return begin_page +"-" +end_page+" ";
    }
}
