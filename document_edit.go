package snd

import (
	"fmt"
)

// Find returns the position of the first sound with the given
// (group, sample) key, in file order. The second result is false when no
// sound has it.
func (d *Document) Find(group, sample int) (int, bool) {
	for i, s := range d.sounds {
		if s.group == group && s.sample == sample {
			return i, true
		}
	}
	return 0, false
}

// validateNewAudio applies the rule for audio handed to the editing
// operations. It must be a RIFF/WAVE blob of plain PCM, 8-bit or 16-bit,
// with at least one channel and a non-zero sample rate — exactly what
// DecodeV1Sound and DecodeV2Sound can read back. Anything else is rejected
// and never converted: converting needs a resampler and codecs that this
// stdlib-only, playback-free library does not carry, so the caller converts
// before handing the audio in.
func validateNewAudio(wav []byte) error {
	sound, err := decodeWAVPCMWithContext(wav, "new audio")
	if err != nil {
		return err
	}
	if sound.SampleRate <= 0 {
		return fmt.Errorf("snd: new audio: invalid sample rate %d", sound.SampleRate)
	}
	return nil
}

// AddSound appends a sound with the given key after the last one. The
// audio must follow the rule of validateNewAudio and the key must not
// already be in the document. Gaps and bytes after the last sound stay
// where they were; they now follow the new sound.
func (d *Document) AddSound(group, sample int, wav []byte) error {
	if err := validateNewAudio(wav); err != nil {
		return fmt.Errorf("adding sound (group %d, sample %d): %w", group, sample, err)
	}
	if _, exists := d.Find(group, sample); exists {
		return fmt.Errorf("snd: sound (group %d, sample %d) already exists", group, sample)
	}

	added := documentSound{group: group, sample: sample, payload: append([]byte{}, wav...), next: nextEnd}
	if n := len(d.sounds); n > 0 {
		last := &d.sounds[n-1]
		added.next, added.literal = last.next, last.literal
		last.next, last.literal = nextFollows, 0
	}
	d.sounds = append(d.sounds, added)
	return nil
}

// ReplaceSound replaces the audio of the sound with the given key, the
// audio following the rule of validateNewAudio. It fails when no sound has
// the key. When several have it, the first one is replaced.
func (d *Document) ReplaceSound(group, sample int, wav []byte) error {
	if err := validateNewAudio(wav); err != nil {
		return fmt.Errorf("replacing sound (group %d, sample %d): %w", group, sample, err)
	}
	i, ok := d.Find(group, sample)
	if !ok {
		return fmt.Errorf("snd: no sound (group %d, sample %d) to replace", group, sample)
	}
	return d.SetPayload(i, wav)
}

// RemoveSound removes the sound with the given key. It fails when no sound
// has the key. When several have it, the first one is removed. The bytes
// between the removed sound and the next one go with it; the header
// comment and the bytes after the last sound stay.
func (d *Document) RemoveSound(group, sample int) error {
	i, ok := d.Find(group, sample)
	if !ok {
		return fmt.Errorf("snd: no sound (group %d, sample %d) to remove", group, sample)
	}

	removed := d.sounds[i]
	if i == len(d.sounds)-1 && i > 0 {
		// The sound before it becomes the last: it takes over how the
		// file ended, and has no next sound left for a gap to lead to.
		prev := &d.sounds[i-1]
		prev.next, prev.literal, prev.gap = removed.next, removed.literal, nil
	}
	d.sounds = append(d.sounds[:i], d.sounds[i+1:]...)

	if len(d.sounds) == 0 {
		// Nothing left for the header to point at: what sat between the
		// header fields and the first sound is now plain trailing data.
		d.trailing = append(d.padding, d.trailing...)
		d.padding = []byte{}
		d.firstLiteral = 0
	}
	return nil
}
