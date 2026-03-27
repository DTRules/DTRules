/**
 * Copyright 2024 Paul Snow
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
 */
package com.dtrules.interpreter;

import org.junit.Test;
import static org.junit.Assert.*;

import com.dtrules.infrastructure.RulesException;

/**
 * Unit tests for RBoolean class.
 */
public class RBooleanTest {

    @Test
    public void testGetRBooleanTrue() throws RulesException {
        RBoolean b = RBoolean.getRBoolean(true);
        assertNotNull("RBoolean should not be null", b);
        assertTrue("booleanValue should return true", b.booleanValue());
    }

    @Test
    public void testGetRBooleanFalse() throws RulesException {
        RBoolean b = RBoolean.getRBoolean(false);
        assertNotNull("RBoolean should not be null", b);
        assertFalse("booleanValue should return false", b.booleanValue());
    }

    @Test
    public void testSingletonTrue() {
        RBoolean b1 = RBoolean.getRBoolean(true);
        RBoolean b2 = RBoolean.getRBoolean(true);
        assertSame("True should be singleton", b1, b2);
    }

    @Test
    public void testSingletonFalse() {
        RBoolean b1 = RBoolean.getRBoolean(false);
        RBoolean b2 = RBoolean.getRBoolean(false);
        assertSame("False should be singleton", b1, b2);
    }

    @Test
    public void testStringValueTrue() {
        RBoolean b = RBoolean.getRBoolean(true);
        assertEquals("true should have stringValue 'true'", "true", b.stringValue());
    }

    @Test
    public void testStringValueFalse() {
        RBoolean b = RBoolean.getRBoolean(false);
        assertEquals("false should have stringValue 'false'", "false", b.stringValue());
    }

    @Test
    public void testEqualsTrue() throws RulesException {
        RBoolean b1 = RBoolean.getRBoolean(true);
        RBoolean b2 = RBoolean.getRBoolean(true);
        assertTrue("true equals true", b1.equals(b2));
    }

    @Test
    public void testEqualsFalse() throws RulesException {
        RBoolean b1 = RBoolean.getRBoolean(false);
        RBoolean b2 = RBoolean.getRBoolean(false);
        assertTrue("false equals false", b1.equals(b2));
    }

    @Test
    public void testEqualsDifferent() throws RulesException {
        RBoolean b1 = RBoolean.getRBoolean(true);
        RBoolean b2 = RBoolean.getRBoolean(false);
        assertFalse("true should not equal false", b1.equals(b2));
    }
}
