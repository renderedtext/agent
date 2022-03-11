#!/bin/ruby
# rubocop:disable all

require_relative '../../e2e'

start_job <<-JSON
  {
    "id": "#{$JOB_ID}",

    "executor": "dockercompose",

    "compose": {
      "containers": [
        {
          "name": "main",
          "image": "ruby:2.6"
        }
      ]
    },

    "env_vars": [],

    "files": [
      { "path": "test.txt", "content": "#{`echo "hello" | base64 | tr -d '\n'`}", "mode": "0644" },
      { "path": "/a/b/c",   "content": "#{`echo "hello" | base64 | tr -d '\n'`}", "mode": "0644" },
      { "path": "/tmp/a",   "content": "#{`echo "hello" | base64 | tr -d '\n'`}", "mode": "0600" }
    ],

    "commands": [
      { "directive": "cat test.txt" },
      { "directive": "cat /a/b/c" },
      { "directive": "stat -c '%a' /tmp/a" }
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

  {"event":"cmd_started",  "timestamp":"*", "directive":"Pulling docker images..."}
  *** LONG_OUTPUT ***
  {"event":"cmd_finished", "timestamp":"*", "directive":"Pulling docker images...","event":"cmd_finished","exit_code":0,"finished_at":"*","started_at":"*","timestamp":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"Starting the docker image..."}
  {"event":"cmd_output",   "timestamp":"*", "output":"Starting a new bash session.\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"Starting the docker image...","event":"cmd_finished","exit_code":0,"finished_at":"*","started_at":"*","timestamp":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"Exporting environment variables"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"Exporting environment variables","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"Injecting Files"}
  {"event":"cmd_output",   "timestamp":"*", "output":"Injecting test.txt with file mode 0644\\n"}
  {"event":"cmd_output",   "timestamp":"*", "output":"Injecting /a/b/c with file mode 0644\\n"}
  {"event":"cmd_output",   "timestamp":"*", "output":"Injecting /tmp/a with file mode 0600\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"Injecting Files","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"cat test.txt"}
  {"event":"cmd_output",   "timestamp":"*", "output":"hello\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"cat test.txt","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"cat /a/b/c"}
  {"event":"cmd_output",   "timestamp":"*", "output":"hello\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"cat /a/b/c","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"stat -c '%a' /tmp/a"}
  {"event":"cmd_output",   "timestamp":"*", "output":"600\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"stat -c '%a' /tmp/a","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"Exporting environment variables"}
  {"event":"cmd_output",   "timestamp":"*", "output":"Exporting SEMAPHORE_JOB_RESULT\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"Exporting environment variables","exit_code":0,"started_at":"*","finished_at":"*"}

  {"event":"job_finished", "timestamp":"*", "result":"passed"}
LOG
