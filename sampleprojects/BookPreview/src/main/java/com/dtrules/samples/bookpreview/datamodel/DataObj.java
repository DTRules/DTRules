package com.dtrules.samples.bookpreview.datamodel;

import com.dtrules.mapping.DataMap;

/**
 * This is a data interface that all data objects need to implement in order
 * to be used by the stand alone application framework
 * @author paulsn
 *
 */
public interface DataObj {
    abstract public void write2DataMap(DataMap datamap);
    
    public int getId();
    
    public boolean getPrinted();
}
