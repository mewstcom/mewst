# typed: strict
# frozen_string_literal: true

require_relative "boot"

require "rails"
# Pick the frameworks you want:
require "active_model/railtie"
require "active_job/railtie"
require "active_record/railtie"
# require "active_storage/engine"
require "action_controller/railtie"
require "action_mailer/railtie"
# require "action_mailbox/engine"
# require "action_text/engine"
require "action_view/railtie"
# require "action_cable/engine"
# require "rails/test_unit/railtie"

# Require the gems listed in Gemfile, including any gems
# you've limited to :test, :development, or :production.
Bundler.require(*Rails.groups)

module Mewst
  class Application < Rails::Application
    # Initialize configuration defaults for originally generated Rails version.
    config.load_defaults 7.1

    # Please, add to the `ignore` list any other `lib` subdirectories that do
    # not contain `.rb` files, or that should not be reloaded or eager loaded.
    # Common ones are `templates`, `generators`, or `middleware`, for example.
    config.autoload_lib(ignore: %w[assets tasks])

    # Configuration for the application, engines, and railties goes here.
    #
    # These settings can be overridden in specific environments using the files
    # in config/environments, which are processed later.
    #
    # config.time_zone = "Central Time (US & Canada)"
    # config.eager_load_paths << Rails.root.join("extras")

    # Don't generate system test files.
    config.generators.system_tests = nil

    # -----------------------------------------------------------------------------------------
    # ここから独自の設定
    # -----------------------------------------------------------------------------------------

    config.active_job.queue_adapter = :good_job

    config.active_record.schema_format = :sql

    config.i18n.available_locales = %i[en ja]

    config.mewst = config_for(:mewst)

    config.middleware.insert_before(Rack::Runtime, Rack::Rewrite) do
      maintenance_file = Rails.public_path.join("maintenance.html")
      send_file(/(.*)$(?<!maintenance|favicons)/, maintenance_file.to_s, if: proc { |rack_env|
        ip_address = rack_env["HTTP_CF_CONNECTING_IP"]

        File.exist?(maintenance_file) &&
          ENV["MEWST_MAINTENANCE_MODE"] == "on" &&
          ip_address != ENV["MEWST_ADMIN_IP"]
      })
    end
  end
end
