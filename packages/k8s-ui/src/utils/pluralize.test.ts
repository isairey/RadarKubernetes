import { describe, it, expect } from 'vitest'
import { englishPlural, pluralNoun, pluralize } from './pluralize'

describe('englishPlural', () => {
  describe('default +s', () => {
    it('handles common nouns', () => {
      expect(englishPlural('cluster')).toBe('clusters')
      expect(englishPlural('cert')).toBe('certs')
      expect(englishPlural('chart')).toBe('charts')
    })
  })

  describe('s/x/ch/sh → +es', () => {
    it('handles s ending', () => {
      expect(englishPlural('class')).toBe('classes')
      expect(englishPlural('ingress')).toBe('ingresses')
    })

    it('handles x ending', () => {
      expect(englishPlural('box')).toBe('boxes')
    })

    it('handles ch ending', () => {
      expect(englishPlural('match')).toBe('matches')
    })

    it('handles sh ending', () => {
      expect(englishPlural('brush')).toBe('brushes')
    })
  })

  describe('consonant-y → -y +ies', () => {
    it('handles policy', () => {
      expect(englishPlural('policy')).toBe('policies')
    })

    it('handles directory', () => {
      expect(englishPlural('directory')).toBe('directories')
    })

    it('preserves vowel-y → +s (does not strip)', () => {
      expect(englishPlural('day')).toBe('days')
      expect(englishPlural('boy')).toBe('boys')
    })
  })

  describe('case insensitivity in detection', () => {
    // The detection regex is case-insensitive (recognizes Class as ending
    // in 's'-class, POLICY as consonant-y) but the rule appends lowercase
    // suffixes. In practice all callers normalize to lowercase before
    // calling — kindToPlural() lowercases its input; pluralize() takes
    // user-controlled nouns which are typically lowercase.
    it('detects rule-applicable suffixes regardless of case', () => {
      expect(englishPlural('Class')).toBe('Classes')   // +es appended lowercase
      expect(englishPlural('Box')).toBe('Boxes')
    })
  })
})

describe('pluralNoun', () => {
  it('returns singular for n===1', () => {
    expect(pluralNoun(1, 'cluster')).toBe('cluster')
  })

  it('returns plural for n!==1', () => {
    expect(pluralNoun(0, 'cluster')).toBe('clusters')
    expect(pluralNoun(2, 'cluster')).toBe('clusters')
    expect(pluralNoun(99, 'cluster')).toBe('clusters')
  })

  it('uses englishPlural rules by default', () => {
    expect(pluralNoun(2, 'policy')).toBe('policies')
    expect(pluralNoun(2, 'class')).toBe('classes')
  })

  it('honors explicit override for irregular plurals', () => {
    expect(pluralNoun(2, 'index', 'indices')).toBe('indices')
    expect(pluralNoun(1, 'index', 'indices')).toBe('index')
  })
})

describe('pluralize', () => {
  it('formats count + noun', () => {
    expect(pluralize(0, 'cluster')).toBe('0 clusters')
    expect(pluralize(1, 'cluster')).toBe('1 cluster')
    expect(pluralize(5, 'cluster')).toBe('5 clusters')
  })

  it('handles irregular plurals via override', () => {
    expect(pluralize(3, 'index', 'indices')).toBe('3 indices')
  })

  it('applies English rules without override', () => {
    expect(pluralize(2, 'policy')).toBe('2 policies')
    expect(pluralize(2, 'class')).toBe('2 classes')
    expect(pluralize(2, 'box')).toBe('2 boxes')
  })
})
