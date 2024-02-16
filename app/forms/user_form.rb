# typed: strict
# frozen_string_literal: true

class UserForm < ApplicationForm
  attribute :locale, :string
  attribute :time_zone, :string

  validates :locale, inclusion: {in: I18n.available_locales.map(&:to_s)}, presence: true
  validates :time_zone, inclusion: {in: ActiveSupport::TimeZone.all.map { |tz| tz.tzinfo.name }}, presence: true
end
