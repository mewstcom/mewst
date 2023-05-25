# typed: strict
# frozen_string_literal: true

class Internal::Mutations::SignUp < Internal::Mutations::Base
  argument :atname, String, required: true
  argument :email, String, required: true
  argument :password, String, required: true

  field :oauth_access_token, Internal::Types::Objects::OauthAccessToken, null: true
  field :errors, [Internal::Types::Objects::ClientErrorType], null: false

  def resolve(atname:, email:, password:)
    form = Forms::SignUp.new(atname:, email:, locale: "ja", password:)

    if form.invalid?
      return {
        oauth_access_token: nil,
        errors: form.errors.full_messages.map { |message| {message:} }
      }
    end

    result = Commands::SignUp.new(form:).call
    result.oauth_access_token.account.track_sign_in

    {
      oauth_access_token: result.oauth_access_token,
      errors: []
    }
  end
end
