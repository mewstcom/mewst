# typed: strict
# frozen_string_literal: true

module Localizable
  extend T::Sig
  extend ActiveSupport::Concern

  sig { params(action: Proc).returns(T.untyped) }
  def set_locale(&action)
    session[:current_locale] = current_account&.locale.presence || params[:locale].presence || session[:current_locale].presence || preferred_locale
    I18n.with_locale(session[:current_locale], &action)
  end

  private

  sig { returns(Symbol) }
  def preferred_locale
    preferred_languages = http_accept_language.user_preferred_languages
    # Chrome returns "ja", but Safari would return "ja-JP", not "ja".
    (preferred_languages.any? { |lang| lang.match?(/ja/) }) ? :ja : :en
  end
end
