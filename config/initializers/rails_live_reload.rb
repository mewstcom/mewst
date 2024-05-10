# typed: strict
# frozen_string_literal: true

if defined?(RailsLiveReload)
  RailsLiveReload.configure do |config|
    config.url = "/rails/live/reload"

    # Default watched folders & files
    config.watch %r{app/components/.+\.(erb|rb)$}, reload: :always
    config.watch %r{app/views/.+\.erb$}, reload: :on_change
    config.watch %r{(app|vendor)/(assets|javascript)/\w+/(.+\.(css|js|html|png|jpg|ts)).*}, reload: :always
    config.watch %r{app/helpers/.+\.rb}, reload: :always
    config.watch %r{lib/mewst/.+\.(erb|rb)$}, reload: :always
    config.watch %r{config/locales/.+\.yml}, reload: :always

    config.enabled = Rails.env.development?
  end
end
