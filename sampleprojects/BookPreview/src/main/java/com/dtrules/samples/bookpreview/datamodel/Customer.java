package com.dtrules.samples.bookpreview.datamodel;

import java.util.ArrayList;
import java.util.List;

import com.dtrules.xmlparser.XMLPrinter;

public class Customer extends ABookObj {
    List<Open_Book> open_books = new ArrayList<Open_Book>();
    List<String>    notes      = new ArrayList<String>();
    public List<Open_Book> getOpen_books() {
        return open_books;
    }
    public void setOpen_books(List<Open_Book> open_books) {
        this.open_books = open_books;
    }
    public List<String> getNotes() {
        return notes;
    }
    public void setNotes(List<String> notes) {
        this.notes = notes;
    }
    
    public void print(XMLPrinter xout){
        xout.opentag("customer","id",id);
            if(printed){
                xout.closetag();
                return;
            }
            xout.opentag("open_books");
                for(Open_Book ob : open_books){
                    ob.print(xout);
                }
            xout.closetag();
        xout.closetag();
    }
    
}
