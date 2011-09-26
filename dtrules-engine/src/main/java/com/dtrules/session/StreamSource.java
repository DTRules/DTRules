package com.dtrules.session;

import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.InputStream;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.URLConnection;

/**
 * The default implementation of 
 * @author paulsn
 *
 */
public class StreamSource implements IStreamSource {

	/**
	 * Our default openfile implementation.  It first attempts to open the
	 * streamSpecification as is.  then it looks at the file path then the
	 * resource path.
	 * 
	 */
	@Override
	public InputStream openStreamSearch (FileType type, RuleSet ruleSet, String streamSpecification) {
		InputStream s = null;
		
		s = openstream(type, streamSpecification);
        if(s!=null)return s;

        if(ruleSet != null){
            s = openstream(type, ruleSet.getFilepath()+"/"+streamSpecification);
	        if(s!=null)return s;
	        s = openstream(type, ruleSet.getResourcepath()+streamSpecification);
        }
	    
        return s;
	}	
    
    /**
     * We attempt to open the streamname as a resource in our jar. Then failing that, we attempt to open it as a URL.
     * Then failing that, we attempt to open it as a file.
     * 
     * @param tag
     *            A String that identifies what sort of file it is to be opened. If you are using some program to
     *            encrypt your decision tables for instance, then this will let you know if you need to decrypt the
     *            source or not.
     * 
     * @param streamname
     *            The name of the file/resource to use for the stream.
     * 
     * @return
     */
	@Override
    public InputStream openstream(FileType tag, String streamname) {
        // First try and open the stream as a resource
        // InputStream s = System.class.getResourceAsStream(streamname);

        InputStream s = getClass().getResourceAsStream(streamname);

        if (s != null)
            return s;

        // If that fails, try and open it as a URL
        try {
            URL url = new URL(streamname);
            URLConnection urlc = url.openConnection();
            s = urlc.getInputStream();
            if (s != null)
                return s;
        } catch (MalformedURLException e) {
        } catch (Exception e) {
        }

        // If that fails, try and open it as a file.
        try {
            s = new FileInputStream(streamname);
            return s;
        } catch (FileNotFoundException e) {
        }

        // If all these fail, return a null.
        return null;

    }
    
    
}
