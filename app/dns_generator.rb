module Joyride
  class DnsGenerator

    protected attr_reader :template, :log, :dnsmasq_process

    public 

    def initialize(log)
      @log = log
      @template = Template.new("/etc/dnsmasq.d/hosts", "/app/templates/dnsmasq.hosts.erb", log)

      # write out basic dnsmasq.conf
      Template.new("/etc/dnsmasq.conf", "/app/templates/dnsmasq.conf.erb", log).write_template()

      #start dnsmasq
      log.info "Starting dnsmasq..."
      # @dnsmasq_process = fork { exec "/usr/sbin/dnsmasq" }
      if ENV.key?('DNS_OPTION' == "true")
        @dnsmasq_process = fork { exec "/usr/sbin/dnsmasq --filter-AAAA" }
      else
        @dnsmasq_process = fork { exec "/usr/sbin/dnsmasq" }
      end
    end

    def process(context)
      log.info "Generating dnsmasq config with hosts:"

      context.domains.each do |domain|
        log.info "\ttemplate => #{domain} #{ENV['HOSTIP']}"
      end

      template.write_template({domains: context.domains})

      log.info "Signaling dnsmasq to reload configuration... "
      Process.kill("HUP", dnsmasq_process)
    end
  end
end