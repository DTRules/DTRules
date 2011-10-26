package com.dtrules.samples.bookpreview.datamodel;

import java.util.ArrayList;
import java.util.List;

import com.dtrules.xmlparser.XMLPrinter;

public class Book extends ABookObj {
    Publisher       publisher;
    int             pages;              // Number of pages in this book
    List<Chapter>   chapters            = new ArrayList<Chapter>();
    List<Chapter>   excluded_chapters   = new ArrayList<Chapter>();
    int             day_limit;
    
    public Publisher getPublisher() {
        return publisher;
    }
    public void setPublisher(Publisher publisher) {
        this.publisher = publisher;
    }
    public List<Chapter> getChapters() {
        return chapters;
    }
    public void setChapters(List<Chapter> chapter) {
        this.chapters = chapter;
    }
    public List<Chapter> getExcluded_chapters() {
        return excluded_chapters;
    }
    public void setExcluded_chapters(List<Chapter> excluded_chapters) {
        this.excluded_chapters = excluded_chapters;
    }
    public int getDay_limit() {
        return day_limit;
    }
    public void setDay_limit(int day_limit) {
        this.day_limit = day_limit;
    }
    public int getPages() {
        return pages;
    }
    public void setPages(int pages) {
        this.pages = pages;
    }
    
    public void print(XMLPrinter xout){
        xout.opentag("book", "id", id);
            if(printed){
                xout.closetag();
                return;
            }

            publisher.print(xout);
            xout.opentag("chapters");
                for(Chapter c : chapters){
                    c.print(xout);
                }
            xout.closetag();
            xout.opentag("excluded_chapters");
                for(Chapter c : excluded_chapters){
                    c.print("excluded_chapter", xout);
                }
            xout.closetag();
       xout.closetag();
       
    }
    
}
