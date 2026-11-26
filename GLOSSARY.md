# vmt

vmt turns recordings made in Apple's Voice Memos app into text transcripts that can be read, copied, and edited outside the app.

## Recordings

**Voice Memos**:
Apple's macOS app that owns the recordings. vmt reads them and never changes them.
_Avoid_: using "voice memo" for a single recording

**Recording**:
One audio recording in Voice Memos, identified by its audio file name. The identity does not change when the recording is renamed, but it is lost when the recording is deleted.
_Avoid_: voice memo, memo, file

**Title**:
The name a Recording shows in Voice Memos. It is for display only and never identifies the Recording.
_Avoid_: name

## Transcription

**Engine**:
The external speech-to-text service that turns a Recording into a Transcript.
_Avoid_: provider, STT, backend

**Model**:
A variant of an Engine. Its price per hour of audio is fixed per Model.
_Avoid_: version

**Cost Estimate**:
The expected price of a Transcription Job: the Recording's length times the Model's hourly price. It is not what the Engine bills.
_Avoid_: cost, price

**Transcription Job**:
One run that sends a Recording to the Engine and writes the resulting Outputs.
_Avoid_: transcription (for the act), task

**Re-transcription**:
A Transcription Job on a Recording that is already Transcribed. It replaces every Output of that Recording, including any edits made to them.
_Avoid_: retry, refresh

**Watch**:
vmt's long-running mode that starts a Transcription Job for each Pending Recording that appears while it runs. It never touches a Transcribed Recording and does not catch up on Recordings that appeared while it was not running.
_Avoid_: auto-transcribe, daemon

**Transcript**:
What the Engine returns for one Recording: an ordered sequence of Segments.
_Avoid_: transcription (for the result), text

**Segment**:
One stretch of speech in a Transcript, with its start time, the words spoken, and its Speaker when Diarization is on.
_Avoid_: line, utterance

**Speaker**:
A voice told apart from the others within one Recording. It does not identify a person, and the same label in two Recordings does not mean the same voice.
_Avoid_: person, participant

**Diarization**:
The Engine splitting a Transcript's Segments by Speaker.
_Avoid_: speaker detection, speaker labels

## Outputs

**Format**:
One of the text representations a Transcript can be written in (txt, md, json, csv, xml).
_Avoid_: type, extension

**Preferred Format**:
The Format that stands for a Transcript when a person reads it: txt when enabled, otherwise the first enabled Format.
_Avoid_: default format, main format

**Output**:
One file holding a Recording's Transcript in a single Format. One Transcription Job writes an Output for every enabled Format.
_Avoid_: transcription (for the file), result file

**Orphan Output**:
An Output whose Recording no longer exists in Voice Memos. vmt leaves it in place.

## Status

**Transcribed**:
A Recording for which at least one Output exists in the current output location, in one of the currently enabled Formats.
_Avoid_: done, complete

**Pending**:
A Recording that is not Transcribed.
_Avoid_: untranscribed, new

**Failed**:
The outcome of a Transcription Job that did not write its Outputs. It belongs to the Job, not to the Recording, and is not remembered after vmt exits.
