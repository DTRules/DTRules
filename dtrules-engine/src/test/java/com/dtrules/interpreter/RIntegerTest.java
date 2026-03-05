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
 * Unit tests for RInteger class.
 */
public class RIntegerTest {

    @Test
    public void testGetRIntegerValue() throws RulesException {
        RInteger i = RInteger.getRIntegerValue(42);
        assertNotNull("RInteger should not be null", i);
        assertEquals("intValue should return 42", 42, i.intValue());
        assertEquals("longValue should return 42", 42L, i.longValue());
    }

    @Test
    public void testGetRIntegerValueZero() {
        RInteger i = RInteger.getRIntegerValue(0);
        assertNotNull("RInteger should not be null", i);
        assertEquals("intValue should return 0", 0, i.intValue());
    }

    @Test
    public void testGetRIntegerValueNegative() {
        RInteger i = RInteger.getRIntegerValue(-42);
        assertNotNull("RInteger should not be null", i);
        assertEquals("intValue should return -42", -42, i.intValue());
    }

    @Test
    public void testGetRIntegerValueLarge() throws RulesException {
        long large = 1000000000L;
        RInteger i = RInteger.getRIntegerValue(large);
        assertNotNull("RInteger should not be null", i);
        assertEquals("longValue should return large value", large, i.longValue());
    }

    @Test
    public void testDoubleValue() throws RulesException {
        RInteger i = RInteger.getRIntegerValue(42);
        assertEquals("doubleValue should return 42.0", 42.0, i.doubleValue(), 0.001);
    }

    @Test
    public void testStringValue() {
        RInteger i = RInteger.getRIntegerValue(42);
        assertEquals("stringValue should return '42'", "42", i.stringValue());
    }

    @Test
    public void testEqualsTrue() throws RulesException {
        RInteger i1 = RInteger.getRIntegerValue(42);
        RInteger i2 = RInteger.getRIntegerValue(42);
        assertTrue("Equal integers should be equal", i1.equals(i2));
    }

    @Test
    public void testEqualsFalse() throws RulesException {
        RInteger i1 = RInteger.getRIntegerValue(42);
        RInteger i2 = RInteger.getRIntegerValue(43);
        assertFalse("Different integers should not be equal", i1.equals(i2));
    }

    @Test
    public void testCompareLess() throws RulesException {
        RInteger i1 = RInteger.getRIntegerValue(10);
        RInteger i2 = RInteger.getRIntegerValue(20);
        assertTrue("10 should be less than 20", i1.compare(i2) < 0);
    }

    @Test
    public void testCompareGreater() throws RulesException {
        RInteger i1 = RInteger.getRIntegerValue(20);
        RInteger i2 = RInteger.getRIntegerValue(10);
        assertTrue("20 should be greater than 10", i1.compare(i2) > 0);
    }

    @Test
    public void testCompareEqual() throws RulesException {
        RInteger i1 = RInteger.getRIntegerValue(42);
        RInteger i2 = RInteger.getRIntegerValue(42);
        assertEquals("Equal integers should compare equal", 0, i1.compare(i2));
    }

    @Test
    public void testCachingSpecialValues() {
        // RInteger caches -1, 0, and 1 specifically
        RInteger z1 = RInteger.getRIntegerValue(0);
        RInteger z2 = RInteger.getRIntegerValue(0);
        assertSame("Zero should be cached", z1, z2);

        RInteger o1 = RInteger.getRIntegerValue(1);
        RInteger o2 = RInteger.getRIntegerValue(1);
        assertSame("One should be cached", o1, o2);

        RInteger m1 = RInteger.getRIntegerValue(-1);
        RInteger m2 = RInteger.getRIntegerValue(-1);
        assertSame("Minus one should be cached", m1, m2);
    }
}
