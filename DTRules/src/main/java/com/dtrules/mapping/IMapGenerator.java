/** 
 * Copyright 2004-2011 DTRules.com, Inc.
 * 
 * See http://DTRules.com for updates and documentation for the DTRules Rules Engine  
 *   
 * Licensed under the Apache License, Version 2.0 (the "License");  
 * you may not use this file except in compliance with the License.  
 * You may obtain a copy of the License at  
 *   
 *      http://www.apache.org/licenses/LICENSE-2.0  
 *   
 * Unless required by applicable law or agreed to in writing, software  
 * distributed under the License is distributed on an "AS IS" BASIS,  
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  
 * See the License for the specific language governing permissions and  
 * limitations under the License.  
 **/

package com.dtrules.mapping;

import java.io.InputStream;

import com.dtrules.xmlparser.XMLPrinter;

/**
 * Interface for the MapGenerator
 * @author ps24876
 *
 */
public interface IMapGenerator {

    public abstract void generateMapping(String mapping, String inputfile,
            String outputfile) throws Exception;

    /**
     * Given an EDD XML, makes a good stab at generating a Mapping file for a given
     * mapping source.  The mapping source is specified as an input in the input column
     * in the EDD. If all your names in the EDD match the names used by your data source,
     * then you are pretty good to go with this output.  If there are some name 
     * differences, then you will need to patch them up by hand... Just looking at the
     * EDD, the code cannot tell what names you are using in your data source.
     * 
     * @param mapping
     * @param input
     * @param out
     * @throws Exception
     */
    public abstract void generateMapping(String mapping, InputStream input,
            XMLPrinter out) throws Exception;

}