# typed: true
# frozen_string_literal: true

class Internal::Accounts::CreateController < Internal::ApplicationController
  def call
    form = Internal::AccountForm.new(
      atname: params[:atname],
      email: params[:email],
      locale: I18n.locale,
      password: params[:password]
    )

    if form.invalid?
      resources = Latest::FormErrorResource.from_errors(errors: form.errors)
      return render(
        json: Latest::ResponseErrorSerializer.new(resources),
        status: :unprocessable_entity
      )
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
