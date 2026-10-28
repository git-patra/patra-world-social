import test from 'node:test';
import assert from 'node:assert/strict';
import {passwordLengthError} from '../src/password-validation.js';

test('password form counts Unicode code points and keeps whitespace',()=>{
 assert.equal(passwordLengthError('😀'.repeat(65)),null);
 assert.match(passwordLengthError('😀'.repeat(129)),/12–128/);
 assert.equal(passwordLengthError(' '.repeat(12)),null);
 assert.match(passwordLengthError(' '.repeat(11)),/12–128/);
});
