/** Count Unicode code points, matching Go's utf8.RuneCountInString server rule. */
export function passwordLengthError(password){
 const count=[...password].length;
 return count>=12&&count<=128?null:'Kata sandi harus 12–128 karakter.';
}
