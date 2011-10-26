package com.dtrules.samples.bookpreview.datamodel;

import java.util.ArrayList;
import java.util.List;

import com.dtrules.mapping.DataMap;


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
    
    public void write2DataMap(DataMap datamap){
        datamap.opentag(this,"book");
        datamap.readDO(this,"book"); {
            if(printed){
                datamap.closetag();
                return;
            }
            publisher.write2DataMap(datamap);
            datamap.opentag("chapters");{
                for(Chapter c : chapters){
                    c.write2DataMap(datamap);
                }
            } datamap.closetag();
        
            datamap.opentag("excluded_chapters");{
                for(Chapter c : excluded_chapters){
                    c.print(datamap,"excluded_chapter");
                }
            }datamap.closetag();
        }datamap.closetag();
       
    }
    
}
