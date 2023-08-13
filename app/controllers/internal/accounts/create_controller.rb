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
      resources = Internal::FormErrorResource.build_from_errors(errors: form.errors)
      return render(
        json: Internal::ResponseErrorSerializer.new(resources),
        status: :unprocessable_entity
      )
    end

    input = CreateAccountService::Input.from_internal_form(form:)
    result = ActiveRecord::Base.transaction do
      r = CreateAccountService.new.call(input:)
      r.oauth_access_token.user!.track_sign_in
      r
    end

    resource = Internal::AccountResource.new(
      oauth_access_token: result.oauth_access_token,
      profile: result.profile,
      user: result.user
    )
    render(
      json: Internal::AccountSerializer.new(resource),
      status: :created
    )
  end
end
