# typed: false
# frozen_string_literal: true

require "capybara/rails"
require "capybara/rspec"
require "capybara/cuprite"

# Docker Composeで立ち上げたChromeに接続するための設定
# ref: https://evilmartians.com/chronicles/system-of-a-test-setting-up-end-to-end-rails-testing
REMOTE_CHROME_URL = ENV["MEWST_CHROME_URL"]
REMOTE_CHROME_HOST, REMOTE_CHROME_PORT =
  if REMOTE_CHROME_URL
    URI.parse(REMOTE_CHROME_URL).yield_self do |uri|
      [uri.host, uri.port]
    end
  end

remote_chrome =
  begin
    if REMOTE_CHROME_URL.nil?
      false
    else
      Socket.tcp(REMOTE_CHROME_HOST, REMOTE_CHROME_PORT, connect_timeout: 1).close
      true
    end
  rescue Errno::ECONNREFUSED, Errno::EHOSTUNREACH, SocketError
    false
  end

remote_options = remote_chrome ? { url: REMOTE_CHROME_URL } : {}

Capybara.register_driver(:mewst_cuprite) do |app|
  Capybara::Cuprite::Driver.new(
    app,
    **{
      browser_options: remote_chrome ? { "no-sandbox" => nil } : {},
      inspector: true,
      window_size: [1200, 800]
    }.merge(remote_options)
  )
end

Capybara.default_driver = :mewst_cuprite
Capybara.javascript_driver = :mewst_cuprite

Capybara.server_host = "0.0.0.0"
Capybara.app_host = "http://#{`hostname`.strip&.downcase || '0.0.0.0'}"

RSpec.configure do |config|
  config.prepend_before(:each, type: :system) do
    driven_by Capybara.javascript_driver
  end

  config.before(:each, type: :system) do
    Capybara.current_session.driver.add_headers("Accept-Language" => "ja")
  end
end
