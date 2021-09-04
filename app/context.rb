module Joyride
  HostnameLabel = "joyride.host.name"
  class Context
    attr_reader :current_container, :running_containers, :current_container, :mutex, :updated_at, :domains, :log

    def initialize(log)
      @mutex = Mutex.new
      @log = log

      update()
    end

    def process_event(event)
      @updated_at = Time.now.to_i

      # we are only intereseted in container start,stop and die events
      # and those containers must have a HostnameLabel
      return unless event.type.eql?("container") && ["start", "stop", "die"].include?(event.action) && event.actor.attributes.has_key?(HostnameLabel)

      # return if we already know about starting container
      return if event.action.eql?("start") && running_containers.any?{|c| c.id.eql?(event.id) }

      # return if we are not removing a stopping container
      return if (event.action.eql?("stop") || event.action.eql?("die")) && running_containers.none?{|c| c.id.eql?(event.id) }
      
      # if we made it here, we need an update
      update()
    rescue => e
      self.log.error e.message
    end

    def reset; @dirty = false end
    def dirty?; @dirty end

    private 

    def update()
      @domains = Docker::Container.all(all: true, filters: { status: ["running"], label: ["HostnameLabel"]  }.to_json)
        .map{|c| c.info["Labels"][HostnameLabel] }
        .uniq

      @dirty = true
    end
  end
end
