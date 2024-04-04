# typed: false
# frozen_string_literal: true

module RequestHelpers
  def sign_in(actor, password: "passw0rd")
    post(sign_in_path, params: {session_form: {email: actor.email, password:}})
    expect(cookies[Session::COOKIE_KEY]).not_to be_nil
  end
end

RSpec.configure do |config|
  config.include RequestHelpers, type: :request
end
