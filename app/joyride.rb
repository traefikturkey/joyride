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

# Auto-detect HOSTIP if not provided (works in host network mode)
def detect_host_ip(log)
  return ENV['HOSTIP'] if ENV['HOSTIP'] && !ENV['HOSTIP'].strip.empty?

  detected_ip = `ip route get 1 2>/dev/null | head -1 | awk '{print $7}'`.strip
  if detected_ip.empty?
    log.error "HOSTIP not set and auto-detection failed. DNS entries will be malformed."
    log.error "Set HOSTIP environment variable or ensure container runs in host network mode."
    return nil
  end

  log.info "Auto-detected HOSTIP: #{detected_ip}"
  detected_ip
end

locator = {}

log = Logger.new($stdout)
log.formatter = proc { |severity, datetime, progname, msg| "#{msg}\n" }

# Set HOSTIP from auto-detection if not already set
ENV['HOSTIP'] = detect_host_ip(log) unless ENV['HOSTIP'] && !ENV['HOSTIP'].strip.empty?

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
  log.info "Joyride has ended!"
end

scheduler.join()
