package com.dtrules.samples.bookpreview.datamodel;

import java.text.SimpleDateFormat;
import java.util.Date;

import com.dtrules.mapping.DataMap;
import com.dtrules.xmlparser.XMLPrinter;

public class Request extends ABookObj {
    Customer customer;
    Book     book;
    String   access;
    boolean  allow;
    int      page_number;
    Date     current_date;
    
    public Customer getCustomer() {
        return customer;
    }
    public void setCustomer(Customer customer) {
        this.customer = customer;
    }
    public Book getBook() {
        return book;
    }
    public void setBook(Book book) {
        this.book = book;
    }
    public String getAccess() {
        return access;
    }
    public void setAccess(String access) {
        this.access = access;
    }
    public boolean isAllow() {
        return allow;
    }
    public void setAllow(boolean allow) {
        this.allow = allow;
    }
    public int getPage_number() {
        return page_number;
    }
    public void setPage_number(int page_number) {
        this.page_number = page_number;
    }
    public Date getCurrent_date() {
        return current_date;
    }
    public void setCurrent_date(Date current_date) {
        this.current_date = current_date;
    }
    
    public void write2DataMap(DataMap datamap){
        
        SimpleDateFormat sdf           = new SimpleDateFormat("MM/dd/yyyy");
        
        datamap.opentag(this,"request");
            if(printed){
                datamap.closetag();
                return;
            }
            datamap.readDO(this, "request");
            book.write2DataMap(datamap);
            customer.write2DataMap(datamap);
        datamap.closetag();
    }
}
