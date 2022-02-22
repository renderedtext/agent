#!/bin/ruby
# rubocop:disable all

require_relative '../../e2e'

start_job <<-JSON
  {
    "id": "#{$JOB_ID}",

    "env_vars": [],

    "files": [],

    "commands": [
      { "directive": "echo Hello World" }
    ],

    "epilogue_always_commands": [
      { "directive": "echo Hello Epilogue" }
    ],

    "epilogue_on_pass_commands": [
      { "directive": "echo Hello On Pass Epilogue" }
    ],

    "epilogue_on_fail_commands": [
      { "directive": "echo Hello On Fail Epilogue" }
    ],

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

  {"event":"cmd_started",  "timestamp":"*", "directive":"echo Hello World"}
  {"event":"cmd_output",   "timestamp":"*", "output":"Hello World\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"echo Hello World","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"Exporting environment variables"}
  {"event":"cmd_output",   "timestamp":"*", "output":"Exporting SEMAPHORE_JOB_RESULT\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"Exporting environment variables","exit_code":0,"started_at":"*","finished_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"echo Hello Epilogue"}
  {"event":"cmd_output",   "timestamp":"*", "output":"Hello Epilogue\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"echo Hello Epilogue","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"cmd_started",  "timestamp":"*", "directive":"echo Hello On Pass Epilogue"}
  {"event":"cmd_output",   "timestamp":"*", "output":"Hello On Pass Epilogue\\n"}
  {"event":"cmd_finished", "timestamp":"*", "directive":"echo Hello On Pass Epilogue","exit_code":0,"finished_at":"*","started_at":"*"}

  {"event":"job_finished", "timestamp":"*", "result":"passed"}
LOG
