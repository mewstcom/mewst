# typed: false
# frozen_string_literal: true

module RequestHelpers
  def sign_in(actor, password: "passw0rd")
    post(sign_in_path, params: {session_form: {email: actor.email, password:}})
    expect(cookies[Session::COOKIE_KEY]).not_to be_nil
  end

  def set_session(session_attrs = {})
    post(
      test_session_path,
      params: {session_attrs:}
    )
    expect(response).to have_http_status(:created)

    session_attrs.each_key do |key|
      expect(session[key]).to be_present
    end
  end
end

RSpec.configure do |config|
  config.include RequestHelpers, type: :request
end
