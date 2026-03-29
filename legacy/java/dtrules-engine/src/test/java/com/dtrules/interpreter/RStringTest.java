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
 * Unit tests for RString class.
 */
public class RStringTest {

    @Test
    public void testNewRString() {
        RString s = RString.newRString("hello");
        assertNotNull("RString should not be null", s);
        assertEquals("stringValue should return 'hello'", "hello", s.stringValue());
    }

    @Test
    public void testNewRStringEmpty() {
        RString s = RString.newRString("");
        assertNotNull("RString should not be null", s);
        assertEquals("stringValue should return empty string", "", s.stringValue());
    }

    @Test
    public void testNewRStringWithSpaces() {
        RString s = RString.newRString("hello world");
        assertEquals("Should preserve spaces", "hello world", s.stringValue());
    }

    @Test
    public void testIntValue() throws RulesException {
        RString s = RString.newRString("42");
        assertEquals("intValue should parse '42' as 42", 42, s.intValue());
    }

    @Test
    public void testLongValue() throws RulesException {
        RString s = RString.newRString("1000000000");
        assertEquals("longValue should parse large number", 1000000000L, s.longValue());
    }

    @Test
    public void testDoubleValue() throws RulesException {
        RString s = RString.newRString("3.14");
        assertEquals("doubleValue should parse '3.14'", 3.14, s.doubleValue(), 0.001);
    }

    @Test
    public void testEqualsTrue() throws RulesException {
        RString s1 = RString.newRString("hello");
        RString s2 = RString.newRString("hello");
        assertTrue("Equal strings should be equal", s1.equals(s2));
    }

    @Test
    public void testEqualsFalse() throws RulesException {
        RString s1 = RString.newRString("hello");
        RString s2 = RString.newRString("world");
        assertFalse("Different strings should not be equal", s1.equals(s2));
    }

    @Test
    public void testCompareLess() throws RulesException {
        RString s1 = RString.newRString("abc");
        RString s2 = RString.newRString("def");
        assertTrue("'abc' should be less than 'def'", s1.compare(s2) < 0);
    }

    @Test
    public void testCompareGreater() throws RulesException {
        RString s1 = RString.newRString("xyz");
        RString s2 = RString.newRString("abc");
        assertTrue("'xyz' should be greater than 'abc'", s1.compare(s2) > 0);
    }

    @Test
    public void testCompareEqual() throws RulesException {
        RString s1 = RString.newRString("hello");
        RString s2 = RString.newRString("hello");
        assertEquals("Equal strings should compare equal", 0, s1.compare(s2));
    }

    @Test
    public void testSpecialCharacters() {
        RString s = RString.newRString("hello\nworld\ttab");
        assertEquals("Should preserve special characters", "hello\nworld\ttab", s.stringValue());
    }

    @Test
    public void testUnicode() {
        RString s = RString.newRString("\u00e9\u00e8\u00ea");
        assertEquals("Should handle unicode", "\u00e9\u00e8\u00ea", s.stringValue());
    }
}
