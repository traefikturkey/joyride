module Joyride
  class Template
    protected attr_accessor :log, :output_path, :template_path

    public

    def initialize(output_path, template_path, log = Logger.new(stdout))
      self.log = log
      self.output_path = output_path
      self.template_path = template_path
    end

    def write_template(template_context={})
      erb = ERB.new(File.open(template_path).read, 0, "<>")
      output = erb.result(TemplateContext.new(template_context).get_binding)
      File.write(output_path, output)
    rescue => ex
      log.error "Error writing to #{template_path}! #{ex.message}\n\tBacktrace - #{ex.backtrace.join("\n\tBacktrace - ")}"
    end
  end

  class TemplateContext
    def initialize(hash)
      hash.each do |key, value|
        singleton_class.send(:define_method, key) { value }
      end 
    end
    
    def ENV
      ENV
    end

    def get_binding
      binding
    end
  end
  
end