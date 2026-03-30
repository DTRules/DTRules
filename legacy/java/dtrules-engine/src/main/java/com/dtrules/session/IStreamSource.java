package com.dtrules.session;

import java.io.InputStream;

/**
 * This interface allows the user to inject into the rules engine
 * logic for finding and returning an input source to use to load
 * rule sets and other artifacts.
 * 
 * @author paulsn
 *
 */
public interface IStreamSource {   
    
    public static enum FileType {
        DECISIONTABLES, EDD, DTRULES_CONFIG, MAP, AUTO_MAP, OTHER;
    }
    
    /**
     * OpenStreamSearch first attempts to open the stream as presented.  Then
     * it does a search of other possible locations for the file.  The
     * first stream that opens successfully is the stream returned.  See
     * the implementation for details.
	 * 
	 * @param tag Identifies the type of file we are trying to open.
	 * @param streamSpecification The resource, URL, or file specification
	 * @return
	 */
	public InputStream openStreamSearch(FileType type, RuleSet ruleSet, String streamSpecification);

	/**
	 * Open a stream of the given type.  In the default implementation, this
	 * function attempts to access a URL, then a resource in a jar file, then 
	 * a filename.  See the implementation for details.
	 * 
	 * @param tag
	 * @param object
	 * @param streamname
	 * @return
	 */
    public InputStream openstream(FileType type, String streamname);


}
