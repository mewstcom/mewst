# typed: true
# frozen_string_literal: true

class Internal::Accounts::CreateController < Internal::ApplicationController
  include Latest::FormErrorable

  def call
    form = Internal::AccountForm.new(
      atname: params[:atname],
      email: params[:email],
      locale: I18n.locale,
      password: params[:password]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    result = CreateAccountUseCase.new.call(
      atname: form.atname.not_nil!,
      email: form.email.not_nil!,
      locale: form.locale.not_nil!,
      password: form.password.not_nil!
    )
    result.user.track_sign_in

    resource = Internal::AccountResource.new(
      user: result.user,
      profile: result.profile,
      oauth_access_token: result.oauth_access_token
    )
    render(
      json: Internal::AccountSerializer.new(resource),
      status: :created
    )
  end
end
