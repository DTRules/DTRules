package com.dtrules.session;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Date;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RBoolean;
import com.dtrules.interpreter.RDate;
import com.dtrules.interpreter.RDouble;
import com.dtrules.interpreter.RInteger;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RTable;
import com.dtrules.interpreter.RType;

public class ComputeDefaultValue implements IComputeDefaultValue {

    public IRObject computeDefaultValue(
            IRSession      session, 
            EntityFactory  ef, 
            String         defaultstr, 
            RType          type) throws RulesException {
                
        if(defaultstr==null ) defaultstr="";
        defaultstr = defaultstr.trim();
        if(defaultstr.equalsIgnoreCase("null")) defaultstr="";
        
        int itype;
        if (type == null){
            itype = IRObject.iNull;
        }else{
            itype = type.getId();
        }
        
        if(itype == IRObject.iEntity) {
                if(defaultstr.length()==0)return RNull.getRNull();
                IREntity e = ef.findcreateRefEntity(false,RName.getRName(defaultstr));
                if(e==null)throw new RulesException(
                        "ParsingError",
                        "EntityFactory.computeDefaultValue()",
                        "Entity Factory does not define an entity '"+defaultstr+"'");
                return e;
        }
        if(itype == IRObject.iArray) {
                if(defaultstr.length()==0) return RArray.newArray(session, true,false);
                RArray rval;
                try{
                     RArray v = (RArray) RString.compile(session, defaultstr, false);     // We assume any values are surrounded by brackets, and regardless make
                     
                     rval = v.get(0).getNonExecutable().rArrayValue();             // sure they are non-executable.
                }catch(RulesException e){
                    throw new RulesException("ParsingError","EntityFactory.computeDefaultValue()","Bad format for an array. \r\n"+
                            "\r\nWe tried to interpret the string \r\n'"+defaultstr+"'\r\nas an array, but could not.\r\n"+e.toString());
                }
                return rval;
        }
        if(itype == IRObject.iString){
                if(defaultstr.length()==0)return RNull.getRNull();
                return RString.newRString(defaultstr);
        }
        if(itype == IRObject.iName){
                if(defaultstr.length()==0)return RNull.getRNull();
                return RName.getRName(defaultstr.trim(),false);
        }
        if(itype == IRObject.iBoolean) {
                if(defaultstr.length()==0)return RNull.getRNull();
                return RBoolean.getRBoolean(defaultstr);
        }   
        if(itype == IRObject.iDouble ) {
                if(defaultstr.length()==0)return RNull.getRNull();
                double value = Double.parseDouble(defaultstr);
                return RDouble.getRDoubleValue(value);
        }   
        if(itype == IRObject.iInteger ) {
                if(defaultstr.length()==0)return RNull.getRNull();
                long value = Long.parseLong(defaultstr);
                return RInteger.getRIntegerValue(value);
        }   
        if(itype == IRObject.iDate ) {
                if(defaultstr.length()==0) return RNull.getRNull();
                SimpleDateFormat fmt = new SimpleDateFormat("MM/dd/yyyy");
                try {
                    Date date = fmt.parse(defaultstr);
                    return RDate.getRTime(date);
                } catch (ParseException e) {
                    throw new RulesException("Invalid Date Format","EntityFactory.computeDefaultValue","Only support dates in 'MM/dd/yyyy' form.");
                }
        }
        if(itype == IRObject.iTable ) {
                RTable table = RTable.newRTable(ef, null, defaultstr);
                if(defaultstr.length()==0) return table;
                table.setValues(session, defaultstr);
                return table;
        }
        
        return RNull.getRNull();
        
    }
}
