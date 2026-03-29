package com.dtrules.session;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RType;

public interface IComputeDefaultValue {

    public IRObject computeDefaultValue(
            IRSession      session, 
            EntityFactory  ef, 
            String         defaultstr, 
            RType          type) throws RulesException;
}