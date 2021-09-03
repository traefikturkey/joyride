module Joyride
  module Generator
    class Base
      protected

      attr_accessor :locator, :output_path, :template_path

      def log
        locator[:log]
      end

      public

      def initialize(output_path, template_path, locator)
        self.locator = locator
        self.output_path = output_path
        self.template_path = template_path
      end

      def write_template(template_context)
        erb = ERB.new(File.open(template_path).read, 0, "<>")
        output = erb.result(template_context)
        file = File.open(output_path, "w")
        file.puts(output)
      rescue => exception
        log.error "Error writing to #{template_path}! #{exception.message}"
      ensure
        self.template_hash = current_hash()
        file.close unless file.nil?
      end

      def dirty?
        (template_hash.nil? || template_hash != current_hash())
      end

      def process(context)
        return false
      end

    end
  end
end