module Joyride
  class DnsGenerator < Generator::Base
    def initialize(locator)
      super("/etc/dnsmasq.d/hosts", "/app/templates/dnsmasq.hosts.erb", locator, 3)
    end

    def process(context)
      log.info "Generating dnsmasq config with hosts:"

      context.domains.each do |domain|
        log.info "\ttemplate => #{domain}"
      end

      write_template(TemplateContext.new({domains: context.domains}).instance_eval { binding })
    end

    class TemplateContext < OpenStruct
      def ENV
        ENV
      end
    end

  end
end