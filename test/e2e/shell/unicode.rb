#!/bin/ruby

Encoding.default_external = Encoding::UTF_8

# rubocop:disable all

require_relative '../../e2e'

#
# This is regression test that verifies that we are correctly processing outgoing
# bytes from the shell.
#
# In case the byte processing is incorrect, the output can contain question mark
# characters instead of valid UTF-8 characters.
#
# Initially, this test only contined Japanese characters to verify that the issue
# has been fixed. However, a sub-case of this issue still caused an issue for a
# customer who is displaying box drawing characters in their tests.
#

start_job <<-JSON
  {
    "id": "#{$JOB_ID}",

    "env_vars": [],
    "files": [],

    "commands": [
      { "directive": "echo 特定の伝説に拠る物語の由来については諸説存在し。特定の伝説に拠る物語の由来については諸説存在し。特定の伝説に拠る物語の由来については諸説存在し。" },
      { "directive": "echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" }
    ],

    "epilogue_always_commands": [],

    "callbacks": {
      "finished": "#{finished_callback_url}",
      "teardown_finished": "#{teardown_callback_url}"
    },
    "logger": #{$LOGGER}
  }
JSON

wait_for_job_to_finish

assert_job_log <<-LOG
  {"event":"job_started",  "timestamp":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"Exporting environment variables"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"Exporting environment variables","exit_code":0,"finished_at":"*","started_at":"*"}
  {"event":"cmd_started",  "timestamp":"*", "directive":"Injecting Files"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"Injecting Files","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"echo 特定の伝説に拠る物語の由来については諸説存在し。特定の伝説に拠る物語の由来については諸説存在し。特定の伝説に拠る物語の由来については諸説存在し。"}
  {"event":"cmd_output",   "timestamp":"*", "output":"特定の伝説に拠る物語の由来については諸説存在し。特定の伝説に拠る物"}
  {"event":"cmd_output",   "timestamp":"*", "output":"語の由来については諸説存在し。特定の伝説に拠る物語の由来については"}
  {"event":"cmd_output",   "timestamp":"*", "output":"諸説存在し。\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"echo 特定の伝説に拠る物語の由来については諸説存在し。特定の伝説に拠る物語の由来については諸説存在し。特定の伝説に拠る物語の由来については諸説存在し。","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"}
  {"event":"cmd_output",   "timestamp":"*", "output":"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"}
  {"event":"cmd_output",   "timestamp":"*", "output":"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"}
  {"event":"cmd_output",   "timestamp":"*", "output":"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"}
  {"event":"cmd_output",   "timestamp":"*", "output":"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"}
  {"event":"cmd_output",   "timestamp":"*", "output":"━━━━━━━━━━━━━━━━━━━━━━\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"Exporting environment variables"}
  {"event":"cmd_output",   "timestamp":"*", "output":"Exporting SEMAPHORE_JOB_RESULT\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"Exporting environment variables","exit_code":0,"started_at":"*","finished_at":"*"}

  {"event":"job_finished", "timestamp":"*", "result":"passed"}
LOG
