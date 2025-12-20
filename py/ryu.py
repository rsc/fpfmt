import math

def compute_shortest(a, b, c, accept_smaller, accept_larger):
	i = 0
	if not accept_larger:
		c -= 1
	all_a_zero = True
	while a//10 < c//10:
		all_a_zero = all_a_zero and a%10 == 0
		a, c = a//10, c//10
		i += 1
	if accept_smaller and all_a_zero:
		while a%10 == 0:
			a, c = a//10, c//10
			i += 1
	return c, i

def compute_shortest2(a, b, c, accept_smaller, accept_larger, break_tie_down):
	i = 0
	if not accept_larger:
		c -= 1
	digit = 0
	all_a_zero = True
	all_b_zero = True
	while a//10 < c//10:
		all_a_zero = all_a_zero and a%10 == 0
		a, c = a//10, c//10
		digit = b%10
		all_b_zero = all_b_zero and b%10 == 0
		b = b//10
		i += 1
	if accept_smaller and all_a_zero:
		while a%10 == 0:
			a, c = a//10, c//10
			digit = b%10
			all_b_zero = all_b_zero and b%10 == 0
			b = b//10
			i += 1
	is_tie = digit == 5 and all_b_zero
	want_round_down = digit < 5 or (is_tie and break_tie_down)
	round_down = (want_round_down and (a != b or all_a_zero)) or (b+1 > c)
	if not round_down:
		b += 1
	return b, i

def table_gte(k, q):
	return 1 + (2**k) // (5**q)

def table_lt(k, e):
	return (5**e) // (2**k)

def format(f):
	# Step 1, p. 271.
	fr, exp = math.frexp(f)
	m = int(fr*(1<<53))
	e = exp - 53

	# Step 2, p. 271.
	e -= 2
	u, v, w = 4*m-2, 4*m, 4*m+2
	if m == 1<<53:
		u += 1

	# Step 3', p. 279.
	if e >= 0:
		q = max(0, math.floor(e*math.log10(2)) - 1)
		k = 128 + math.floor(q*math.log2(5))
		a = (u * table_gte(k, q)) >> (-e+q+k)
		b = (v * table_gte(k, q)) >> (-e+q+k)
		c = (w * table_gte(k, q)) >> (-e+q+k)
		za = u%(5**q) == 0
		zb = v%(5**(q-1)) == 0 # TODO really q-1?
		zc = w%(5**q) == 0
	else:
		q = max(0, math.floor(-e*log(2, base=5))-1)
		k = math.ceil(q*math.log2(5)) - 128
		a = (u * table_lt(k, -e2-q)) >> (q-k)
		b = (v * table_lt(k, -e2-q)) >> (q-k)
		c = (w * table_lt(k, -e2-q)) >> (q-k)
		za = u%(2**q) == 0
		zb = u%(2**(q-1)) == 0
		zc = u%(2**q) == 0

	# Step 4', p. 279.
	break_tie_down = zb  # TODO not in paper!
	accept_smaller = m%2 == 0
	accept_larger = m%2 == 0

	d, p = compute_shortest(a, b, c, accept_smaller and za, accept_larger or not zc)
	return d, p + q

if __name__ == '__main__':
	print(format(2**89))

