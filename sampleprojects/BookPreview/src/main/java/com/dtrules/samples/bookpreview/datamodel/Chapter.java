package com.dtrules.samples.bookpreview.datamodel;

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
    
    void print(String tag, XMLPrinter xout){
        xout.opentag(tag, "id",id);
            if(printed){
                xout.closetag();
                return;
            }
    
            xout.printdata("begin_page",begin_page);
            xout.printdata("end_page",end_page);
        xout.closetag();
    }
    
    @Override
    void print(XMLPrinter xout) { print("chapter",xout);}
    
    @Override
    public String toString() {
        return begin_page +"-" +end_page+" ";
    }
}
