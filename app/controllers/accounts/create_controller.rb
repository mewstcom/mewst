# typed: true
# frozen_string_literal: true

class Accounts::CreateController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable
  include EmailConfirmationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_succeeded_email_confirmation

  sig { returns(T.untyped) }
  def call
    @form = AccountForm.new(form_params.merge(email: @email))

    if @form.invalid?
      return render("accounts/new/call", status: :unprocessable_entity)
    end

    result = CreateAccountUseCase.new(client: v1_internal_client).call(form: @form)

    if result.errors
      @form.add_use_case_errors(result.errors.not_nil!)
      return render("accounts/new/call", status: :unprocessable_entity)
    end

    reset_session
    sign_in(result.account.not_nil!.actor.not_nil!)

    flash[:notice] = t("messages.accounts.signed_up_successfully")
    redirect_to home_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:account_form), ActionController::Parameters).permit(:atname, :password)
  end
end
