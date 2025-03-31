#!/usr/bin/env ruby

require 'docker'
require 'erb'
require 'json'
require 'logger'
require 'ostruct'
require 'rufus-scheduler'

require_relative 'context'
require_relative 'template'
require_relative 'dns_generator'

# no buffering output
$stdout.sync = true

locator = {}

log = Logger.new($stdout)
log.formatter = proc { |severity, datetime, progname, msg| "#{msg}\n" }

log.info "Started serf..."
log.debug ENV['HOSTIP']
serf_process = fork do
  ENV["SERF_HANDLER_CONFIG"] = File.join(Dir.pwd, "serf-handlers")  # Set environment variable
  exec "serf agent -profile=wan -join=192.168.16.115 -advertise=#{ENV['HOSTIP']}:7946 -bind=0.0.0.0:7946 -event-handler=/gems/bin/serf-handler"
end

file_path = "/etc/hosts.d/hosts"
begin
  File.new(file_path, "w") if !File.exist?(file_path)
rescue => ex
  log.warn("Error creating file: #{ex.message}")
end

generator = Joyride::DnsGenerator.new(log)
context = Joyride::Context.new(log)

define_singleton_method("log") { log }
define_singleton_method("generator") { generator }

scheduler = Rufus::Scheduler.new

scheduler.every '3s', :first => :now, :mutex => context.mutex do
  Docker::Event.since(context.updated_at, until: Time.now.to_i) {|event| context.process_event(event)}

  if context.dirty?
    generator.process(context)
    context.reset()
  end
end

Kernel.trap( "INT" ) do
  scheduler.shutdown
  Process.kill("TERM", serf_process)
  log.info "Joyride has ended!"
end

scheduler.join()
