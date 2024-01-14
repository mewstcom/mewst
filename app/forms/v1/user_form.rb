# typed: strict
# frozen_string_literal: true

class V1::UserForm < V1::ApplicationForm
  attribute :locale, :string
  attribute :time_zone, :string

  validates :locale, inclusion: {in: I18n.available_locales.map(&:to_s)}, presence: true
  validates :time_zone, inclusion: {in: ActiveSupport::TimeZone.all.map { |tz| tz.tzinfo.name }}, presence: true
end
