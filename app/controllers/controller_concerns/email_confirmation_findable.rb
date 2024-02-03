# typed: strict
# frozen_string_literal: true

module ControllerConcerns::EmailConfirmationFindable
  extend T::Sig
  extend ActiveSupport::Concern

  sig { returns(T.untyped) }
  def require_email_confirmation_id
    unless session[:email_confirmation_id]
      redirect_to root_path
    end
  end

  sig { returns(T.untyped) }
  def require_succeeded_email_confirmation
    unless session[:email_confirmation_id]
      return redirect_to(root_path)
    end

    @email_confirmation = EmailConfirmation.succeeded.find_by(id: session[:email_confirmation_id])

    unless @email_confirmation
      redirect_to root_path
    end
  end
end
