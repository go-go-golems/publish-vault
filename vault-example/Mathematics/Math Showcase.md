---
title: Math Showcase
tags: [math, reference]
---

# Math Showcase

This note exercises every branch of the math pipeline. It doubles as a manual
test fixture: if any section below renders wrongly, the corresponding case in
`internal/parser/math_test.go` is the place to start.

## Inline math

The identity $e^{i\pi} + 1 = 0$ ties together five constants. Subscripted
sequences such as $a_1, a_2, \ldots, a_n$ must keep their underscores rather
than turning into emphasis, and $f * g * h$ must keep its asterisks.

You can also use LaTeX-native delimiters: \(x^2 + y^2 = z^2\).

## Display math

$$
f(x) = \frac{1}{\sigma\sqrt{2\pi}} \, e^{-\frac{1}{2}\left(\frac{x-\mu}{\sigma}\right)^2}
$$

Aligned environments span several lines and use `&` as the alignment
character and `\\` as the line break — both of which plain Markdown would
otherwise destroy:

$$
\begin{align}
\mathbb{E}[X]      &= \mu \\
\mathrm{Var}(X)    &= \sigma^2 \\
\mathrm{Cov}(X, Y) &= \mathbb{E}[(X - \mu_X)(Y - \mu_Y)]
\end{align}
$$

Cases and matrices:

$$
\mathrm{sgn}(x) = \begin{cases}
  -1 & \text{if } x < 0 \\
   0 & \text{if } x = 0 \\
   1 & \text{if } x > 0
\end{cases}
\qquad
A = \begin{pmatrix} a & b \\ c & d \end{pmatrix}
$$

The LaTeX-native display form works too:

\[
\sum_{n=1}^{\infty} \frac{1}{n^2} = \frac{\pi^2}{6}
\]

## What must *not* be treated as math

Prices are prose, not formulas. The hardback costs $30 and the paperback costs
$25; neither dollar sign opens math, because a closing `$` may not be preceded
by whitespace nor followed by a digit. An escaped dollar sign — \$100 — is
also literal.

Math inside code stays code. Inline: `$x^2$`. And in a fence:

```markdown
Inline math is written $x^2$ and display math as $$x^2$$.
```

```python
# A dollar sign in a string is not math either
print(f"total: ${amount}")
```

## Math in other contexts

> [!note] Inside a callout
> Euler's identity, $e^{i\pi} + 1 = 0$, still typesets here.

- In a list item: $\nabla \cdot \mathbf{E} = \frac{\rho}{\varepsilon_0}$
- With a wiki link nearby: see [[Epistemology]] for $P(H \mid E)$

| Symbol | Meaning | Definition |
| --- | --- | --- |
| $\mu$ | mean | $\frac{1}{n}\sum_i x_i$ |
| $\sigma$ | standard deviation | $\sqrt{\mathbb{E}[(X-\mu)^2]}$ |

## Overflow

A deliberately wide equation, which must scroll inside its own box rather than
forcing a horizontal scrollbar onto the page:

$$
\underbrace{a_1 + a_2 + a_3 + a_4 + a_5 + a_6 + a_7 + a_8 + a_9 + a_{10} + a_{11} + a_{12} + a_{13} + a_{14} + a_{15}}_{\text{fifteen terms}} = \sum_{i=1}^{15} a_i
$$
