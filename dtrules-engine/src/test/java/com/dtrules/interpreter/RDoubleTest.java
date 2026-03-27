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
 * Unit tests for RDouble class.
 */
public class RDoubleTest {

    @Test
    public void testGetRDoubleValue() throws RulesException {
        RDouble d = RDouble.getRDoubleValue(3.14);
        assertNotNull("RDouble should not be null", d);
        assertEquals("doubleValue should return 3.14", 3.14, d.doubleValue(), 0.001);
    }

    @Test
    public void testGetRDoubleValueZero() throws RulesException {
        RDouble d = RDouble.getRDoubleValue(0.0);
        assertNotNull("RDouble should not be null", d);
        assertEquals("doubleValue should return 0.0", 0.0, d.doubleValue(), 0.001);
    }

    @Test
    public void testGetRDoubleValueNegative() throws RulesException {
        RDouble d = RDouble.getRDoubleValue(-2.5);
        assertNotNull("RDouble should not be null", d);
        assertEquals("doubleValue should return -2.5", -2.5, d.doubleValue(), 0.001);
    }

    @Test
    public void testIntValue() throws RulesException {
        RDouble d = RDouble.getRDoubleValue(3.7);
        assertEquals("intValue should truncate to 3", 3, d.intValue());
    }

    @Test
    public void testLongValue() throws RulesException {
        RDouble d = RDouble.getRDoubleValue(3.7);
        assertEquals("longValue should truncate to 3", 3L, d.longValue());
    }

    @Test
    public void testStringValue() throws RulesException {
        RDouble d = RDouble.getRDoubleValue(3.14);
        assertNotNull("stringValue should not be null", d.stringValue());
        assertTrue("stringValue should contain 3.14", d.stringValue().contains("3.14"));
    }

    @Test
    public void testEqualsTrue() throws RulesException {
        RDouble d1 = RDouble.getRDoubleValue(3.14);
        RDouble d2 = RDouble.getRDoubleValue(3.14);
        assertTrue("Equal doubles should be equal", d1.equals(d2));
    }

    @Test
    public void testEqualsFalse() throws RulesException {
        RDouble d1 = RDouble.getRDoubleValue(3.14);
        RDouble d2 = RDouble.getRDoubleValue(2.71);
        assertFalse("Different doubles should not be equal", d1.equals(d2));
    }

    @Test
    public void testCompareLess() throws RulesException {
        RDouble d1 = RDouble.getRDoubleValue(1.0);
        RDouble d2 = RDouble.getRDoubleValue(2.0);
        assertTrue("1.0 should be less than 2.0", d1.compare(d2) < 0);
    }

    @Test
    public void testCompareGreater() throws RulesException {
        RDouble d1 = RDouble.getRDoubleValue(2.0);
        RDouble d2 = RDouble.getRDoubleValue(1.0);
        assertTrue("2.0 should be greater than 1.0", d1.compare(d2) > 0);
    }

    @Test
    public void testCompareEqual() throws RulesException {
        RDouble d1 = RDouble.getRDoubleValue(3.14);
        RDouble d2 = RDouble.getRDoubleValue(3.14);
        assertEquals("Equal doubles should compare equal", 0, d1.compare(d2));
    }

    @Test
    public void testLargeValue() throws RulesException {
        double large = 1e15;
        RDouble d = RDouble.getRDoubleValue(large);
        assertEquals("Should handle large values", large, d.doubleValue(), 1e5);
    }

    @Test
    public void testSmallValue() throws RulesException {
        double small = 1e-15;
        RDouble d = RDouble.getRDoubleValue(small);
        assertEquals("Should handle small values", small, d.doubleValue(), 1e-20);
    }
}
