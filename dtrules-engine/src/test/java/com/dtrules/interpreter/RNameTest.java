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
 * Unit tests for RName class.
 * RName is an interned name/symbol type used for identifiers.
 */
public class RNameTest {

    @Test
    public void testGetRName() {
        RName n = RName.getRName("test");
        assertNotNull("RName should not be null", n);
        assertEquals("stringValue should return 'test'", "test", n.stringValue());
    }

    @Test
    public void testGetRNameInterning() {
        RName n1 = RName.getRName("test");
        RName n2 = RName.getRName("test");
        assertSame("Same name should return same instance", n1, n2);
    }

    @Test
    public void testGetRNameDifferent() {
        RName n1 = RName.getRName("foo");
        RName n2 = RName.getRName("bar");
        assertNotSame("Different names should be different instances", n1, n2);
    }

    @Test
    public void testEqualsTrue() throws RulesException {
        RName n1 = RName.getRName("test");
        RName n2 = RName.getRName("test");
        assertTrue("Interned names should be equal", n1.equals(n2));
    }

    @Test
    public void testEqualsFalse() throws RulesException {
        RName n1 = RName.getRName("foo");
        RName n2 = RName.getRName("bar");
        assertFalse("Different names should not be equal", n1.equals(n2));
    }

    @Test
    public void testCompareLess() throws RulesException {
        RName n1 = RName.getRName("aaa");
        RName n2 = RName.getRName("bbb");
        assertTrue("'aaa' should be less than 'bbb'", n1.compare(n2) < 0);
    }

    @Test
    public void testCompareGreater() throws RulesException {
        RName n1 = RName.getRName("zzz");
        RName n2 = RName.getRName("aaa");
        assertTrue("'zzz' should be greater than 'aaa'", n1.compare(n2) > 0);
    }

    @Test
    public void testCompareEqual() throws RulesException {
        RName n1 = RName.getRName("test");
        RName n2 = RName.getRName("test");
        assertEquals("Same names should compare equal", 0, n1.compare(n2));
    }

    @Test
    public void testWithUnderscore() {
        RName n = RName.getRName("test_name");
        assertEquals("Should allow underscores", "test_name", n.stringValue());
    }

    @Test
    public void testWithNumbers() {
        RName n = RName.getRName("test123");
        assertEquals("Should allow numbers", "test123", n.stringValue());
    }

    @Test
    public void testCaseSensitive() {
        RName n1 = RName.getRName("Test");
        RName n2 = RName.getRName("test");
        assertNotSame("Names should be case sensitive", n1, n2);
    }
}
