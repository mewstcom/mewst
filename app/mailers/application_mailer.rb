# typed: strict
# frozen_string_literal: true

class ApplicationMailer < ActionMailer::Base
  extend T::Sig

  default from: "Mewst <no-reply@#{Rails.configuration.mewst["email_domain"]}>"
end
