#!/usr/bin/env ruby

require 'docker'
require 'json'
require 'logger'
require 'rufus-scheduler'

require_relative 'context'
require_relative 'container'
require_relative 'domain'
require_relative 'dns_generator'

# no buffering output
$stdout.sync = true

locator = {}

log = Logger.new($stdout) 
log.formatter = proc { |severity, datetime, progname, msg| "#{msg}\n" }

generator = Joyride::DnsGenerator.new(log) 

context = Joyride::Context.new(log)

define_singleton_method("log") { log }
define_singleton_method("generator") { generator }

scheduler = Rufus::Scheduler.new

scheduler.every '1s', :first => :now, :mutex => context.mutex do
  Docker::Event.since(context.updated_at, until: Time.now.to_i) {|event| context.process_event(event)}
  
  return unless context.dirty?
  
  generator.process(context)
  context.reset()
end

Kernel.trap( "INT" ) do 
  scheduler.shutdown
  log.info "Joyride has ended!"
end

scheduler.join()