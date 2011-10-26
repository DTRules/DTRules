package com.dtrules.samples.bookpreview.datamodel;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;

import com.dtrules.xmlparser.XMLPrinter;

public class Open_Book extends ABookObj{
    Date            begin_date;
    List<Page>      pages               = new ArrayList<Page>();
    List<Chapter>   chapters_viewed     = new ArrayList<Chapter>();
    Book            book;

    public Date getBegin_date() {
        return begin_date;
    }
    public void setBegin_date(Date begin_date) {
        this.begin_date = begin_date;
    }
    public List<Page> getPages() {
        return pages;
    }
    public void setPages(List<Page> pages) {
        this.pages = pages;
    }
    public List<Chapter> getChapters_viewed() {
        return chapters_viewed;
    }
    public void setChapters_viewed(List<Chapter> chapters_viewed) {
        this.chapters_viewed = chapters_viewed;
    }
    public Book getBook() {
        return book;
    }
    public void setBook(Book book) {
        this.book = book;
    }
    
    public void print(XMLPrinter xout){
        
        SimpleDateFormat sdf           = new SimpleDateFormat("MM/dd/yyyy");
      
        xout.opentag("open_book","id",id);
            if(printed){
                xout.closetag();
                return;
            }

            xout.printdata("begin_date",sdf.format(begin_date));
            
            xout.opentag("pages");
                for(Page p : pages){
                    p.print(xout);
                }
            xout.closetag();
            
            xout.opentag("chapters_viewed");
                for(Chapter c : chapters_viewed){
                    c.print("chapter_viewed", xout);
                }
            xout.closetag();
            
            book.print(xout);
       
        xout.closetag();
       
    }
    
}
