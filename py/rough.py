# rough numbers ⟪x⟫

import math

class Rough:
	def __init__(self, rough):
		self.r = rough
	@classmethod
	def value(cls, x):
		return Rough(math.floor(x*4) | (x*4 != math.floor(x*4)))
	def __str__(self):
		return f'{self.r>>2}.{"05"[(self.r>>1)&1]}{"+" if self.r&1 else ""}'
	def __repr__(self):
		return f'{self.r} ‘{str(self)}’'
	def __eq__(self, other):
		return type(other) == Rough and self.r == other.r
	def __truediv__(self, d):  # / operator
		return Rough(self.r//d | (self.r%d != 0) | self.r&1)
	def __rshift__(self, s):  # >> operator
		return Rough(self.r>>s | ((self.r&((1<<s)-1)) != 0) | self.r&1)
	def floor(self):
		return self.r >> 2
	def roundHalfDown(self):
		return (self.r + 1) >> 2
	def round(self):
		return (self.r + 1 + ((self.r>>2)&1)) >> 2
	def roundHalfUp(self):
		return (self.r + 2) >> 2
	def ceil(self):
		return (self.r + 3) >> 2
	def nudge(self, δ):
		return Rough(self.r + δ)

assert str(Rough.value(123)) == '123.0'
assert str(Rough.value(123.5)) == '123.5'
assert str(Rough.value(123.4)) == '123.0+'
assert str(Rough.value(123.6)) == '123.5+'

assert Rough.value(10) / 6 == Rough.value(10/6)
assert Rough.value(10) / 6 == Rough.value(10/6)

assert Rough.value(123)/10 == Rough.value(12.3)
assert Rough.value(123)>>3 == Rough.value(123/8)
