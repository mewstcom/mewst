# typed: true
# frozen_string_literal: true

class Accounts::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable
  include ControllerConcerns::EmailConfirmationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_succeeded_email_confirmation

  sig { returns(T.untyped) }
  def call
    @form = AccountForm.new(
      form_params.merge(
        email: @email_confirmation.not_nil!.email.not_nil!,
        locale: current_locale.serialize,
        time_zone: "Asia/Tokyo" # TODO: あとでユーザのタイムゾーンを指定する
      )
    )

    if @form.invalid?
      return render("accounts/new/call", status: :unprocessable_entity)
    end

    create_account_result = CreateAccountUseCase.new.call(
      atname: @form.atname.not_nil!,
      email: @form.email.not_nil!,
      locale: @form.locale.not_nil!,
      password: @form.password.not_nil!,
      time_zone: @form.time_zone.not_nil!
    )

    create_session_result = CreateSessionUseCase.new.call(
      actor: create_account_result.actor,
      ip_address: request.remote_ip,
      user_agent: request.user_agent
    )

    sign_in(create_session_result.session)

    flash[:notice] = t("messages.accounts.signed_up_successfully")
    redirect_to home_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:account_form), ActionController::Parameters).permit(:atname, :password)
  end
end
