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
 * Unit tests for RArray class.
 * Uses newArrayTraceInterface to create arrays without a session.
 */
public class RArrayTest {

    private static int testId = 1000;

    private RArray createTestArray() throws RulesException {
        return RArray.newArrayTraceInterface(testId++, true, false);
    }

    @Test
    public void testNewArray() throws RulesException {
        RArray arr = createTestArray();
        assertNotNull("RArray should not be null", arr);
        assertEquals("New array should have size 0", 0, arr.size());
    }

    @Test
    public void testAddAndGet() throws RulesException {
        RArray arr = createTestArray();
        RInteger val = RInteger.getRIntegerValue(42);
        arr.add(val);
        assertEquals("Array should have size 1 after add", 1, arr.size());
        assertEquals("get(0) should return added value", val, arr.get(0));
    }

    @Test
    public void testAddMultiple() throws RulesException {
        RArray arr = createTestArray();
        for (int i = 0; i < 5; i++) {
            arr.add(RInteger.getRIntegerValue(i));
        }
        assertEquals("Array should have size 5", 5, arr.size());
        for (int i = 0; i < 5; i++) {
            assertEquals("Values should match", i, arr.get(i).intValue());
        }
    }

    @Test
    public void testInsertAtIndex() throws RulesException {
        RArray arr = createTestArray();
        arr.add(RInteger.getRIntegerValue(10));
        arr.add(RInteger.getRIntegerValue(20));
        // Insert at index 0 shifts existing elements
        arr.add(0, RInteger.getRIntegerValue(100));
        assertEquals("Inserted value should be at index 0", 100, arr.get(0).intValue());
        assertEquals("Original first value should shift to index 1", 10, arr.get(1).intValue());
        assertEquals("Original second value should shift to index 2", 20, arr.get(2).intValue());
    }

    @Test
    public void testDeleteAtIndex() throws RulesException {
        RArray arr = createTestArray();
        arr.add(RInteger.getRIntegerValue(1));
        arr.add(RInteger.getRIntegerValue(2));
        arr.add(RInteger.getRIntegerValue(3));
        arr.delete(1);  // delete at index 1
        assertEquals("Size should be 2 after delete", 2, arr.size());
        assertEquals("First element unchanged", 1, arr.get(0).intValue());
        assertEquals("Third element moved to second position", 3, arr.get(1).intValue());
    }

    @Test
    public void testContainsTrue() throws RulesException {
        RArray arr = createTestArray();
        RInteger val = RInteger.getRIntegerValue(42);
        arr.add(val);
        assertTrue("Array should contain added value", arr.contains(val));
    }

    @Test
    public void testContainsFalse() throws RulesException {
        RArray arr = createTestArray();
        arr.add(RInteger.getRIntegerValue(42));
        RInteger other = RInteger.getRIntegerValue(100);
        assertFalse("Array should not contain non-added value", arr.contains(other));
    }

    @Test
    public void testClear() throws RulesException {
        RArray arr = createTestArray();
        arr.add(RInteger.getRIntegerValue(1));
        arr.add(RInteger.getRIntegerValue(2));
        arr.clear();
        assertEquals("Size should be 0 after clear", 0, arr.size());
    }

    @Test
    public void testMixedTypes() throws RulesException {
        RArray arr = createTestArray();
        arr.add(RInteger.getRIntegerValue(42));
        arr.add(RString.newRString("hello"));
        arr.add(RBoolean.getRBoolean(true));
        assertEquals("Array should have size 3", 3, arr.size());
        assertEquals("First element should be integer", 42, arr.get(0).intValue());
        assertEquals("Second element should be string", "hello", arr.get(1).stringValue());
        assertTrue("Third element should be boolean true", arr.get(2).booleanValue());
    }

    @Test
    public void testStringValue() throws RulesException {
        RArray arr = createTestArray();
        arr.add(RInteger.getRIntegerValue(1));
        arr.add(RInteger.getRIntegerValue(2));
        String s = arr.stringValue();
        assertNotNull("stringValue should not be null", s);
        assertTrue("stringValue should contain '1'", s.contains("1"));
        assertTrue("stringValue should contain '2'", s.contains("2"));
    }
}
